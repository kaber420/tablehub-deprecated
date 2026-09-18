package events

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tablehub/hub/internal/crypto"
	"github.com/tablehub/hub/internal/db"
	"github.com/tablehub/hub/internal/state"
)

var (
	nc  *nats.Conn
	js  nats.JetStreamContext
	sub *nats.Subscription

	// BroadcastCallback se utiliza para reenviar eventos al panel web local sin importar el paquete web directamente.
	BroadcastCallback func(event string, payload []byte)
)

// --- ESTRUCTURAS DE PAYLOADS COMPATIBLES Y SEGUROS ---

type SignedPayload struct {
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"` // Firma Ed25519 en formato hexadecimal
}

type DeviceStatusInnerPayload struct {
	MAC          string `json:"mac"`
	IPAddress    string `json:"ip"`
	BatteryLevel int    `json:"battery"`
	WiFiSignal   int    `json:"wifi"`
	Status       string `json:"status"`
	Timestamp    int64  `json:"timestamp"`
}

type DeviceEventInnerPayload struct {
	MAC       string `json:"mac"`
	EventType string `json:"event_type"` // "alert" | "button_press" | etc.
	Kind      string `json:"kind"`       // "waiter" | "bill" | "help" | etc.
	Timestamp int64  `json:"timestamp"`
}

type ProvisionPayload struct {
	MAC            string `json:"mac"`
	PublicKey      string `json:"public_key"`
	IPAddress      string `json:"ip"`
	Type           string `json:"type"` // "table_pad" | "kitchen_button" etc.
	TableID        string `json:"table_id"`
	BootstrapToken string `json:"bootstrap_token"`
}

// Estructuras clásicas para compatibilidad anterior
type DeviceStatusPayload struct {
	MACAddress   string `json:"mac"`
	TableNumber  string `json:"table"`
	IPAddress    string `json:"ip"`
	BatteryLevel int    `json:"battery"`
	WiFiSignal   int    `json:"wifi"`
	Status       string `json:"status"`
}

type DeviceAlertPayload struct {
	TableNumber string `json:"table"`
	AlertKind   string `json:"kind"` // "waiter" | "bill" | "help"
	Timestamp   int64  `json:"timestamp"`
}

// InitJetStream conecta al servidor local de NATS e inicializa JetStream.
func InitJetStream(natsPort int) error {
	var err error
	var conn *nats.Conn

	// Reintentar la conexión por si el servidor tarda en levantar
	for i := 0; i < 5; i++ {
		conn, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsPort),
			nats.Name("tablehub-client"),
			nats.Timeout(2*time.Second),
			nats.MaxReconnects(5),
			nats.ReconnectWait(2*time.Second),
		)
		if err == nil {
			break
		}
		log.Printf("Esperando a NATS en puerto %d... (reintento %d/5)", natsPort, i+1)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("no se pudo conectar al cliente NATS: %w", err)
	}

	nc = conn

	js, err = nc.JetStream()
	if err != nil {
		nc.Close()
		return fmt.Errorf("error al obtener contexto JetStream: %w", err)
	}

	// Crear/Actualizar el Stream TABLE_EVENTS para escuchar todos los tópicos bajo tablehub.>
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "TABLE_EVENTS",
		Subjects: []string{"tablehub.>"},
		Storage:  nats.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 1 * 1024 * 1024 * 1024, // 1GB
		Replicas: 1,
	})
	if err != nil {
		log.Printf("Advertencia: No se pudo crear/actualizar stream TABLE_EVENTS (%v). Intentando recrearlo...", err)
		_ = js.DeleteStream("TABLE_EVENTS")
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     "TABLE_EVENTS",
			Subjects: []string{"tablehub.>"},
			Storage:  nats.FileStorage,
			MaxAge:   72 * time.Hour,
			MaxBytes: 1 * 1024 * 1024 * 1024,
			Replicas: 1,
		})
		if err != nil {
			nc.Close()
			return fmt.Errorf("error al recrear stream JetStream: %w", err)
		}
	}

	return nil
}

// StartConsumer se suscribe persistente a los eventos generales
func StartConsumer() error {
	if js == nil {
		return fmt.Errorf("jetstream no inicializado")
	}

	var err error
	sub, err = js.QueueSubscribe("tablehub.>", "hub-worker", handleNatsEvent,
		nats.BindStream("TABLE_EVENTS"),
		nats.Durable("hub-worker"),
		nats.ManualAck(),
	)
	if err != nil {
		return fmt.Errorf("error al crear suscripción JetStream: %w", err)
	}

	log.Println("Consumidor JetStream duradero 'hub-worker' iniciado para 'tablehub.>'.")
	return nil
}

func handleNatsEvent(msg *nats.Msg) {
	parts := strings.Split(msg.Subject, ".")
	if len(parts) < 3 {
		log.Printf("NATS: Asunto de mensaje no válido recibido (muy corto): %s", msg.Subject)
		msg.Ack()
		return
	}

	domain := parts[1] // "table" o "device"

	var err error

	// Caso especial: Solicitud de aprovisionamiento de dispositivo sin firma previa
	if domain == "device" && parts[2] == "provision" {
		err = processDeviceProvision(msg.Data)
	} else if domain == "device" && len(parts) >= 4 {
		mac := parts[2]
		eventType := parts[3] // "status" o "event"

		switch eventType {
		case "status":
			err = processDeviceStatusSecure(msg.Data, mac)
		case "event":
			err = processDeviceEventSecure(msg.Data, mac)
		case "lwt":
			err = processDeviceLWT(msg.Data, mac)
		case "bill":
			if len(parts) >= 5 && parts[4] == "request" {
				err = processDeviceBillRequest(msg.Data, mac)
			}
		case "command", "state":
			// Ignorar mensajes de comandos y estado interno enviados a dispositivos
			return
		default:
			log.Printf("NATS: Tipo de evento de dispositivo desconocido: %s", eventType)
		}
	} else if domain == "table" && len(parts) >= 4 {
		// Modo clásico/retrocompatible
		tableNumber := parts[2]
		eventType := parts[3] // "status" o "alert"

		switch eventType {
		case "status":
			err = processDeviceStatusOld(msg.Data, tableNumber)
		case "alert":
			err = processDeviceAlertOld(msg.Data, tableNumber)
		default:
			log.Printf("NATS: Tipo de evento de mesa clásico desconocido: %s", eventType)
		}
	} else {
		log.Printf("NATS: Formato de asunto de mensaje no coincidente: %s", msg.Subject)
	}

	if err != nil {
		log.Printf("NATS: Error procesando asunto %s: %v", msg.Subject, err)
		errStr := err.Error()
		// Los errores de validación de JSON o firmas criptográficas fallidas se descartan (Ack)
		// para evitar bucles interminables en la cola de mensajes.
		if strings.Contains(errStr, "json") || strings.Contains(errStr, "invalid") || strings.Contains(errStr, "signature") || strings.Contains(errStr, "desvío") || strings.Contains(errStr, "no registrado") {
			msg.Ack()
		} else {
			msg.NakWithDelay(3 * time.Second)
		}
	} else {
		msg.Ack()
	}
}

// --- LOGICA DE VERIFICACIÓN CRIPTOGRÁFICA ---

func verifyDeviceSignature(mac string, rawPayload []byte, signatureHex string) error {
	device, err := db.GetDeviceByMAC(mac)
	if err != nil {
		return fmt.Errorf("error al leer base de datos: %w", err)
	}
	if device == nil {
		return fmt.Errorf("dispositivo %s no registrado", mac)
	}
	if device.Status == "blocked" {
		return fmt.Errorf("dispositivo %s bloqueado por el administrador", mac)
	}
	if device.PublicKey == nil || *device.PublicKey == "" {
		return fmt.Errorf("dispositivo %s no posee clave pública de verificación", mac)
	}

	pubKeyBytes, err := crypto.DecodeKeyFromHex(*device.PublicKey)
	if err != nil {
		return fmt.Errorf("clave pública corrupta en DB: %w", err)
	}

	sigBytes, err := crypto.DecodeKeyFromHex(signatureHex)
	if err != nil {
		return fmt.Errorf("firma hexadecimal inválida: %w", err)
	}

	if !crypto.Verify(pubKeyBytes, rawPayload, sigBytes) {
		return fmt.Errorf("firma criptográfica Ed25519 no coincide")
	}

	return nil
}

func verifyTimestamp(msgTimestamp int64) error {
	now := time.Now().Unix()
	diff := now - msgTimestamp
	if diff < 0 {
		diff = -diff
	}
	// Tolerar una desviación máxima de 5 minutos (300 segundos) para evitar ataques de replay
	if diff > 300 {
		return fmt.Errorf("marca de tiempo del mensaje fuera de rango (desvío de %d segundos)", diff)
	}
	return nil
}

// --- PROCESAMIENTO DE DISPOSITIVOS SEGUROS (Ed25519) ---

func processDeviceProvision(data []byte) error {
	var payload ProvisionPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("error unmarshal json provision: %w", err)
	}

	if payload.MAC == "" {
		return fmt.Errorf("datos de aprovisionamiento incompletos (mac vacío)")
	}

	if payload.Type == "" {
		payload.Type = "table_pad"
	}

	log.Printf("NATS JetStream: Solicitud de aprovisionamiento de dispositivo %s (%s, Mesa: %s)", payload.MAC, payload.Type, payload.TableID)

	// Validar bootstrap_token si se incluye en la solicitud
	if payload.BootstrapToken != "" && payload.TableID != "" {
		valid, err := db.ValidateAndConsumeBootstrapToken(payload.TableID, payload.BootstrapToken)
		if err != nil {
			log.Printf("NATS: Error al validar bootstrap token: %v", err)
		}
		if valid {
			log.Printf("NATS: [ÉXITO] Bootstrap token VÁLIDO para Mesa %s! Auto-activando dispositivo %s", payload.TableID, payload.MAC)
			_ = db.DeletePendingDevice(payload.TableID)
			err = db.ApproveDevice(payload.MAC, payload.Type, &payload.TableID, nil, nil)
			if err != nil {
				log.Printf("NATS: Error al auto-aprobar dispositivo: %v", err)
			}

			// Publicar confirmación al dispositivo por MQTT
			respTopic := fmt.Sprintf("tablehub/device/%s/state", payload.MAC)
			respPayload, _ := json.Marshal(map[string]interface{}{
				"status":   "active",
				"table_id": payload.TableID,
				"message":  "Dispositivo activado exitosamente",
			})
			_ = Publish(respTopic, respPayload)

			if BroadcastCallback != nil {
				BroadcastCallback("device.status_update", respPayload)
			}
			return nil
		} else {
			log.Printf("NATS WARN: Bootstrap token no coincide o expiró para Mesa %s", payload.TableID)
		}
	}

	existing, err := db.GetDeviceByMAC(payload.MAC)
	if err != nil {
		return fmt.Errorf("error base de datos: %w", err)
	}

	if existing == nil {
		var tNum *string
		if payload.TableID != "" {
			tNum = &payload.TableID
		}
		_, err = db.DB.Exec(`
			INSERT INTO devices (mac_address, device_type, table_number, ip_address, public_key, status, last_seen)
			VALUES (?, ?, ?, ?, ?, 'unprovisioned', CURRENT_TIMESTAMP)`,
			payload.MAC, payload.Type, tNum, payload.IPAddress, payload.PublicKey)
	} else {
		_, err = db.DB.Exec(`
			UPDATE devices
			SET ip_address = ?, public_key = ?, last_seen = CURRENT_TIMESTAMP
			WHERE mac_address = ?`,
			payload.IPAddress, payload.PublicKey, payload.MAC)
	}
	if err != nil {
		return fmt.Errorf("error al guardar en SQLite: %w", err)
	}

	if BroadcastCallback != nil {
		BroadcastCallback("device.provision_request", data)
	}

	return nil
}

func processDeviceStatusSecure(data []byte, mac string) error {
	var signed SignedPayload
	if err := json.Unmarshal(data, &signed); err != nil {
		return fmt.Errorf("error unmarshal json firmado status: %w", err)
	}

	if err := verifyDeviceSignature(mac, signed.Payload, signed.Signature); err != nil {
		return fmt.Errorf("error firma status: %w", err)
	}

	var inner DeviceStatusInnerPayload
	if err := json.Unmarshal(signed.Payload, &inner); err != nil {
		return fmt.Errorf("error unmarshal json interno status: %w", err)
	}

	if err := verifyTimestamp(inner.Timestamp); err != nil {
		return fmt.Errorf("error timestamp status: %w", err)
	}

	log.Printf("NATS JetStream: Telemetría válida de %s -> Batería: %d%%, Señal: %d dBm",
		mac, inner.BatteryLevel, inner.WiFiSignal)

	err := db.UpdateDeviceStatus(mac, inner.IPAddress, inner.BatteryLevel, inner.WiFiSignal, inner.Status)
	if err != nil {
		return fmt.Errorf("error al actualizar DB: %w", err)
	}

	if BroadcastCallback != nil {
		innerJSON, _ := json.Marshal(inner)
		BroadcastCallback("device.status_update", innerJSON)
	}

	// Resiliencia: Re-publicar órdenes activas desde caché si el dispositivo está activo
	if inner.Status == "active" {
		device, dbErr := db.GetDeviceByMAC(mac)
		if dbErr == nil && device != nil && device.TableNumber != nil && *device.TableNumber != "" {
			ordersList := state.GetOrdersForTable(*device.TableNumber)
			for _, flatOrder := range ordersList {
				var o struct {
					OrderID string `json:"id"`
				}
				if err := json.Unmarshal(flatOrder, &o); err == nil && o.OrderID != "" {
					mqttSubject := fmt.Sprintf("tablehub.device.%s.order.%s.state", mac, o.OrderID)
					_ = Publish(mqttSubject, flatOrder)
					log.Printf("[MQTT/NATS] Re-publicando orden %s para dispositivo reconectado %s (Mesa %s) en subject: %s", o.OrderID, mac, *device.TableNumber, mqttSubject)
				}
			}
		}
	}

	return nil
}

func processDeviceEventSecure(data []byte, mac string) error {
	var signed SignedPayload
	if err := json.Unmarshal(data, &signed); err != nil {
		return fmt.Errorf("error unmarshal json firmado event: %w", err)
	}

	if err := verifyDeviceSignature(mac, signed.Payload, signed.Signature); err != nil {
		return fmt.Errorf("error firma event: %w", err)
	}

	var inner DeviceEventInnerPayload
	if err := json.Unmarshal(signed.Payload, &inner); err != nil {
		return fmt.Errorf("error unmarshal json interno event: %w", err)
	}

	if err := verifyTimestamp(inner.Timestamp); err != nil {
		return fmt.Errorf("error timestamp event: %w", err)
	}

	device, err := db.GetDeviceByMAC(mac)
	if err != nil || device == nil {
		return fmt.Errorf("dispositivo no encontrado al procesar evento: %v", err)
	}

	// Si no está activo (ej: aún está en espera de aprobación), ignoramos eventos de negocio
	if device.Status != "active" {
		return fmt.Errorf("dispositivo %s inactivo (estado: %s)", mac, device.Status)
	}

	loc := "Sin Ubicación"
	if device.LocationName != nil {
		loc = *device.LocationName
	}

	log.Printf("NATS JetStream: Evento verificado de %s (%s - %s) -> Tipo: %s, Kind: %s",
		mac, device.DeviceType, loc, inner.EventType, inner.Kind)

	// Mapeo lógico de eventos según rol del dispositivo
	if device.DeviceType == "table_pad" && device.TableNumber != nil && *device.TableNumber != "" {
		// Mapear al flujo clásico de mesas para compatibilidad de los meseros
		alertPayload := map[string]interface{}{
			"table":     *device.TableNumber,
			"kind":      inner.Kind,
			"timestamp": inner.Timestamp,
		}
		alertJSON, _ := json.Marshal(alertPayload)

		if BroadcastCallback != nil {
			BroadcastCallback("device.alert", alertJSON)
		}

		_, err = db.DB.Exec(`
			INSERT INTO event_logs (event_type, payload) 
			VALUES (?, ?)`,
			"alert."+inner.Kind, string(alertJSON))
		if err != nil {
			return fmt.Errorf("error log table alert insert: %w", err)
		}
	} else {
		// Evento genérico (Cocina, Barra, etc.)
		genericPayload := map[string]interface{}{
			"mac":           mac,
			"device_type":   device.DeviceType,
			"location_name": device.LocationName,
			"event_type":    inner.EventType,
			"kind":          inner.Kind,
			"timestamp":     inner.Timestamp,
		}
		genericJSON, _ := json.Marshal(genericPayload)

		if BroadcastCallback != nil {
			BroadcastCallback("device.generic_event", genericJSON)
		}

		_, err = db.DB.Exec(`
			INSERT INTO event_logs (event_type, payload) 
			VALUES (?, ?)`,
			"device."+inner.EventType, string(genericJSON))
		if err != nil {
			return fmt.Errorf("error log generic event insert: %w", err)
		}
	}

	return nil
}

func processDeviceLWT(data []byte, mac string) error {
	log.Printf("NATS JetStream: LWT (Last Will and Testament) recibido. Dispositivo desconectado de MQTT: %s", mac)
	
	err := db.SetDeviceOffline(mac)
	if err != nil {
		return fmt.Errorf("error al actualizar DB (LWT offline): %w", err)
	}

	if BroadcastCallback != nil {
		payloadJSON, _ := json.Marshal(map[string]interface{}{
			"mac":    mac,
			"status": "offline",
		})
		BroadcastCallback("device.status_update", payloadJSON)
	}
	return nil
}

func processDeviceBillRequest(data []byte, mac string) error {
	device, err := db.GetDeviceByMAC(mac)
	if err != nil || device == nil {
		return fmt.Errorf("dispositivo no encontrado para bill request: %v", err)
	}

	tableNumber := ""
	if device.TableNumber != nil {
		tableNumber = *device.TableNumber
	}

	var items []map[string]interface{}
	subtotal := 0.0

	ordersList := state.GetOrdersForTable(tableNumber)
	for _, flatOrder := range ordersList {
		var o struct {
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
		}
		if err := json.Unmarshal(flatOrder, &o); err == nil {
			for _, it := range o.Items {
				items = append(items, map[string]interface{}{
					"name":  it.Name,
					"qty":   1,
					"price": 100.00, // Precio estimado por ahora
				})
				subtotal += 100.00
			}
		}
	}

	if len(items) == 0 {
		items = []map[string]interface{}{}
		subtotal = 0.0
	}

	tax := subtotal * 0.16
	total := subtotal + tax

	respPayload := map[string]interface{}{
		"subtotal": subtotal,
		"tax":      tax,
		"total":    total,
		"items":    items,
	}

	respJSON, _ := json.Marshal(respPayload)
	topic := fmt.Sprintf("tablehub.device.%s.bill.state", mac)
	_ = Publish(topic, respJSON)
	log.Printf("[MQTT/NATS] Cuenta calculada y publicada para %s (Mesa %s) en %s", mac, tableNumber, topic)

	return nil
}

// --- PROCESADORES RETROCOMPATIBLES ---

func processDeviceStatusOld(data []byte, tableNumber string) error {
	var payload DeviceStatusPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("error json unmarshal: %w", err)
	}

	if payload.TableNumber == "" {
		payload.TableNumber = tableNumber
	}

	log.Printf("NATS JetStream (Clásico): Estado de mesa %s -> Batería: %d%%, WiFi: %d dBm",
		payload.TableNumber, payload.BatteryLevel, payload.WiFiSignal)

	// Intentar actualizar usando la MAC como clave primaria
	mac := payload.MACAddress
	if mac == "" {
		mac = "LEGACY_" + payload.TableNumber
	}

	err := db.UpdateDeviceStatus(mac, payload.IPAddress, payload.BatteryLevel, payload.WiFiSignal, payload.Status)
	if err != nil {
		return fmt.Errorf("error sqlite db update: %w", err)
	}

	if BroadcastCallback != nil {
		BroadcastCallback("device.status_update", data)
	}
	return nil
}

func processDeviceAlertOld(data []byte, tableNumber string) error {
	var payload DeviceAlertPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("error json unmarshal: %w", err)
	}

	if payload.TableNumber == "" {
		payload.TableNumber = tableNumber
	}

	log.Printf("NATS JetStream (Clásico): Alerta mesa %s -> %s", payload.TableNumber, payload.AlertKind)

	if BroadcastCallback != nil {
		BroadcastCallback("device.alert", data)
	}

	_, err := db.DB.Exec(`
		INSERT INTO event_logs (event_type, payload) 
		VALUES (?, ?)`,
		"alert."+payload.AlertKind, string(data))
	if err != nil {
		return fmt.Errorf("error sqlite log insert: %w", err)
	}
	return nil
}

// Publish envía un mensaje a un subject de NATS
func Publish(subject string, data []byte) error {
	if nc == nil {
		return fmt.Errorf("conexión cliente NATS no inicializada")
	}
	return nc.Publish(subject, data)
}

// Subscribe permite suscribirse a un subject usando Core NATS
func Subscribe(subject string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	if nc == nil {
		return nil, fmt.Errorf("conexión cliente NATS no inicializada")
	}
	return nc.Subscribe(subject, handler)
}

// CloseClient cierra las suscripciones y la conexión cliente de NATS.
func CloseClient() {
	if sub != nil {
		log.Println("Drenando consumidor JetStream...")
		sub.Drain()
		sub = nil
	}
	if nc != nil {
		log.Println("Cerrando conexión cliente NATS...")
		nc.Close()
		nc = nil
	}
}

// GetStatus obtiene el estado de ejecución y conexión de NATS/JetStream.
func GetStatus() map[string]interface{} {
	running := false
	if natsServer != nil {
		running = natsServer.Running()
	}

	clientStatus := "not_initialized"
	if nc != nil {
		switch nc.Status() {
		case nats.CONNECTED:
			clientStatus = "connected"
		case nats.CONNECTING:
			clientStatus = "connecting"
		case nats.DISCONNECTED:
			clientStatus = "disconnected"
		case nats.CLOSED:
			clientStatus = "closed"
		case nats.RECONNECTING:
			clientStatus = "reconnecting"
		default:
			clientStatus = "unknown"
		}
	}

	return map[string]interface{}{
		"nats_running": running,
		"nats_client":  clientStatus,
		"mqtt_port":    mqttPort,
		"nats_port":    natsPort,
	}
}
