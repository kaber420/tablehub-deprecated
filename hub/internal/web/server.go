package web

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/tablehub/hub/internal/crypto"
	"github.com/tablehub/hub/internal/db"
	"github.com/tablehub/hub/internal/devices"
	"github.com/tablehub/hub/internal/events"
	"github.com/tablehub/hub/internal/state"
	"github.com/tablehub/hub/pkg/cdn"
)

// Upgrader para las conexiones de WebSocket locales del panel
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Hub representa el gestor de conexiones WebSocket locales (el panel de admin)
type LocalWebHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

var LocalHub = LocalWebHub{
	clients:    make(map[*websocket.Conn]bool),
	broadcast:  make(chan []byte),
	register:   make(chan *websocket.Conn),
	unregister: make(chan *websocket.Conn),
}

func (h *LocalWebHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Println("Cliente del panel web local conectado")
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
				log.Println("Cliente del panel web local desconectado")
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error enviando mensaje a cliente web local: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *LocalWebHub) BroadcastMessage(message []byte) {
	h.broadcast <- message
}

func BroadcastEvent(event string, payload []byte) {
	localMsg, _ := json.Marshal(TunnelMessage{
		Event:     event,
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	})
	LocalHub.BroadcastMessage(localMsg)
}

func HandleLocalWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al upgradear WebSocket local: %v", err)
		return
	}
	LocalHub.register <- conn

	defer func() {
		LocalHub.unregister <- conn
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

type TunnelMessage struct {
	Event     string          `json:"event"`
	Timestamp int64           `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type ChallengePayload struct {
	ChallengeHex string `json:"challenge_hex"`
}

type AuthResponsePayload struct {
	OrganizationID string `json:"organization_id"`
	HubID          string `json:"hub_id"`
	SignatureHex   string `json:"signature_hex"`
}

var (
	cloudCancel        context.CancelFunc
	cloudMu            sync.Mutex
	isCloudConnected   bool
	isCloudConnectedMu sync.RWMutex
	activeCloudConn    *websocket.Conn
	activeCloudConnMu  sync.Mutex
)

func SetCloudConnected(connected bool) {
	isCloudConnectedMu.Lock()
	defer isCloudConnectedMu.Unlock()
	isCloudConnected = connected
}

func IsCloudConnected() bool {
	isCloudConnectedMu.RLock()
	defer isCloudConnectedMu.RUnlock()
	return isCloudConnected
}

func SetActiveCloudConn(conn *websocket.Conn) {
	activeCloudConnMu.Lock()
	defer activeCloudConnMu.Unlock()
	if activeCloudConn != nil && activeCloudConn != conn {
		activeCloudConn.Close()
	}
	activeCloudConn = conn
}

func CloseActiveCloudConn() {
	activeCloudConnMu.Lock()
	if activeCloudConn != nil {
		activeCloudConn.Close()
		activeCloudConn = nil
	}
	activeCloudConnMu.Unlock()
	SetCloudConnected(false)
}

func RestartCloudConnection() {
	cloudMu.Lock()
	defer cloudMu.Unlock()
	CloseActiveCloudConn()
	if cloudCancel != nil {
		cloudCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	cloudCancel = cancel
	go ConnectToCloud(ctx)
}


// ConnectToCloud abre el túnel WebSocket hacia el Servidor SaaS e inicia el flujo Ed25519.
// Implementa reconexión con backoff exponencial: espera inicial 1s, se duplica hasta 60s máximo.
func ConnectToCloud(ctx context.Context) {
	wait := 1 * time.Second

	for {
		select {
		case <-ctx.Done():
			log.Println("Deteniendo túnel de la nube...")
			return
		default:
		}

		cloudURL, _ := db.GetSetting("cloud_url")
		restaurantID, _ := db.GetSetting("restaurant_id")
		cloudEnabledStr, _ := db.GetSetting("cloud_enabled")

		if cloudURL == "" || restaurantID == "" || cloudEnabledStr != "true" {
			log.Println("Conexión a la nube deshabilitada o no configurada. En espera de configuración...")
			select {
			case <-ctx.Done():
				return
			}
		}

		log.Printf("Intentando conectar al túnel de la nube en %s...", cloudURL)

		conn, _, err := websocket.DefaultDialer.Dial(cloudURL, nil)
		if err != nil {
			log.Printf("Error al conectar con la nube: %v. Reintentando en %v...", err, wait)
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
			wait *= 2
			if wait > 60*time.Second {
				wait = 60 * time.Second
			}
			continue
		}

		SetActiveCloudConn(conn)

		wait = 1 * time.Second

		log.Println("Conexión WebSocket establecida con la nube. Iniciando negociación de seguridad...")

		errCh := make(chan error, 1)
		go func() {
			errCh <- handleCloudSession(conn, restaurantID)
		}()

		select {
		case <-ctx.Done():
			CloseActiveCloudConn()
			return
		case err := <-errCh:
			if err != nil {
				log.Printf("Sesión del túnel de la nube cerrada con error: %v", err)
			}
			CloseActiveCloudConn()
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func handleCloudSession(conn *websocket.Conn, restaurantID string) error {
	defer SetActiveCloudConn(nil)
	InitCloudKeys()


	keyFile := filepath.Join("data", "hub_cloud.key")
	privKeyHexBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("no se pudo leer el archivo de clave privada %s: %w", keyFile, err)
	}

	privKeyHex := strings.TrimSpace(string(privKeyHexBytes))
	if privKeyHex == "" {
		return fmt.Errorf("el archivo de clave privada %s está vacío", keyFile)
	}

	privBytes, err := crypto.DecodeKeyFromHex(privKeyHex)
	if err != nil {
		return fmt.Errorf("error decodificando clave privada desde %s: %w", keyFile, err)
	}
	privKey := ed25519.PrivateKey(privBytes)

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("error configurando read deadline para desafío: %w", err)
	}

	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("error leyendo desafío de la nube: %w", err)
	}

	var challengeMsg TunnelMessage
	if err := json.Unmarshal(msgBytes, &challengeMsg); err != nil {
		return fmt.Errorf("error parseando mensaje de desafío: %w", err)
	}

	if challengeMsg.Event != "auth_challenge" {
		return fmt.Errorf("evento inesperado durante handshake: %s", challengeMsg.Event)
	}

	var challengePayload ChallengePayload
	if err := json.Unmarshal(challengeMsg.Payload, &challengePayload); err != nil {
		return fmt.Errorf("error parseando payload del desafío: %w", err)
	}

	log.Printf("Desafío criptográfico recibido de la nube: %s", challengePayload.ChallengeHex)

	challengeBytes, err := hex.DecodeString(challengePayload.ChallengeHex)
	if err != nil {
		return fmt.Errorf("desafío no es un string hex válido: %w", err)
	}

	signature := crypto.Sign(privKey, challengeBytes)
	signatureHex := hex.EncodeToString(signature)

	hubIDStr, _ := db.GetSetting("hub_id")
	responsePayload := AuthResponsePayload{
		OrganizationID: restaurantID,
		HubID:          hubIDStr,
		SignatureHex:   signatureHex,
	}

	payloadBytes, _ := json.Marshal(responsePayload)
	responseMsg := TunnelMessage{
		Event:     "auth_response",
		Timestamp: time.Now().Unix(),
		Payload:   payloadBytes,
	}

	responseBytes, _ := json.Marshal(responseMsg)
	if err := conn.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("error configurando write deadline para respuesta: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, responseBytes); err != nil {
		return fmt.Errorf("error enviando respuesta de autenticación: %w", err)
	}

	log.Println("Respuesta de autenticación Ed25519 enviada a la nube. Esperando confirmación...")

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("error configurando read deadline para confirmación: %w", err)
	}
	_, resultBytes, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("error al recibir confirmación de autenticación: %w", err)
	}

	var resultMsg TunnelMessage
	if err := json.Unmarshal(resultBytes, &resultMsg); err != nil {
		return fmt.Errorf("error parseando mensaje de confirmación: %w", err)
	}

	if resultMsg.Event != "auth_success" {
		return fmt.Errorf("autenticación rechazada por la nube: %s", resultMsg.Event)
	}

	SetCloudConnected(true)
	defer SetCloudConnected(false)

	log.Println("¡Autenticación exitosa! Túnel de comunicación seguro activo.")


	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPingHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(10*time.Second))
	})
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

	for {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("conexión perdida con la nube: %w", err)
		}

		var event TunnelMessage
		if err := json.Unmarshal(data, &event); err != nil {
			log.Printf("Error al decodificar evento de la nube: %v", err)
			continue
		}

		switch event.Event {
		case "ping":
			pongBytes, _ := json.Marshal(TunnelMessage{
				Event:     "pong",
				Timestamp: time.Now().Unix(),
			})
			conn.WriteMessage(websocket.TextMessage, pongBytes)
		}
	}
}

type NormalizedOrderItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	StatusLabel string   `json:"status_label"`
	Options     []string `json:"options,omitempty"`
}

func getProgressFromStatus(status string) int {
	status = strings.ToLower(status)
	if strings.Contains(status, "listo") || strings.Contains(status, "servido") || strings.Contains(status, "completado") || strings.Contains(status, "entregado") || strings.Contains(status, "completed") || strings.Contains(status, "delivered") || strings.Contains(status, "ready") {
		return 100
	}
	if strings.Contains(status, "cocina") || strings.Contains(status, "preparando") || strings.Contains(status, "proceso") || strings.Contains(status, "processing") || strings.Contains(status, "cooking") || strings.Contains(status, "kitchen") {
		return 50
	}
	return 0
}

// ProcessCloudOrder procesa una orden entrante desde NATS Cloud
func ProcessCloudOrder(payload []byte) {
	log.Printf("Nueva orden del POS recibida vía NATS Cloud: %s", string(payload))

	normalized := normalizeOrderPayload(payload)

	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(normalized, &rawData); err != nil {
		log.Printf("Error al deserializar orden de la nube: %v", err)
		return
	}

	// Propagar al UI local
	eventMsg := TunnelMessage{
		Event:     "order.created",
		Timestamp: time.Now().Unix(),
		Payload:   normalized,
	}
	data, _ := json.Marshal(eventMsg)
	LocalHub.BroadcastMessage(data)

	// Extraer información detallada para publicar al dispositivo (ESP32) vía MQTT
	var temp struct {
		Order struct {
			ID               string                `json:"id"`
			TableNumber      string                `json:"table_number"`
			CustomerName     string                `json:"customer_name"`
			OrderProgressPct int                   `json:"order_progress_pct"`
			Items            []NormalizedOrderItem `json:"items"`
		} `json:"order"`
	}

	if err := json.Unmarshal(normalized, &temp); err == nil && temp.Order.ID != "" {
		tbl := temp.Order.TableNumber
		if tbl != "" {
			// Buscar la MAC activa para esa mesa
			device, err := db.GetActiveDeviceByTable(tbl)
			if err == nil && device != nil {
				flatOrder, _ := json.Marshal(temp.Order)

				// Guardar en la caché de órdenes en memoria
				state.SaveOrderToCache(tbl, temp.Order.ID, flatOrder)

				// Publicar directo a MQTT usando el subject NATS mapeado
				mqttSubject := fmt.Sprintf("tablehub.device.%s.order.%s.state", device.MACAddress, temp.Order.ID)
				events.Publish(mqttSubject, flatOrder)
				log.Printf("[MQTT/NATS] Orden %s publicada para el dispositivo %s (Mesa %s) en subject: %s", temp.Order.ID, device.MACAddress, tbl, mqttSubject)
			} else {
				log.Printf("[MQTT/NATS] No se encontró dispositivo activo para la mesa %s, ignorando publicación MQTT directa.", tbl)
			}
		}
	}
}

// extractTableIdentifier extrae el identificador de mesa de un mapa de datos JSON,
// soportando strings planos (table, table_number, table_name, table_id),
// estructuras anidadas como {"table": {"table_name": "Mesa 4", "name": "Mesa 4"}},
// y wrappers como {"order": {"table_number": "Mesa 12"}} o {"data": {...}}.
func extractTableIdentifier(data map[string]json.RawMessage) string {
	for _, wrapper := range []string{"order", "data"} {
		if raw, ok := data[wrapper]; ok && len(raw) > 0 && raw[0] == '{' {
			var nested map[string]json.RawMessage
			if json.Unmarshal(raw, &nested) == nil {
				if tbl := extractTableIdentifier(nested); tbl != "" {
					return tbl
				}
			}
		}
	}

	for _, key := range []string{"table", "table_number", "table_name", "table_id"} {
		if raw, ok := data[key]; ok && len(raw) > 0 && raw[0] == '"' {
			var s string
			if json.Unmarshal(raw, &s) == nil && s != "" {
				return s
			}
		}
	}

	if raw, ok := data["table"]; ok && len(raw) > 0 && raw[0] == '{' {
		var nested struct {
			TableName string `json:"table_name"`
			Name      string `json:"name"`
		}
		if json.Unmarshal(raw, &nested) == nil {
			if nested.TableName != "" {
				return nested.TableName
			}
			if nested.Name != "" {
				return nested.Name
			}
		}
	}

	return ""
}

func extractFieldString(data map[string]interface{}, keys []string) string {
	for _, wrapper := range []string{"order", "data"} {
		if wData, ok := data[wrapper].(map[string]interface{}); ok {
			for _, k := range keys {
				if v, ok := wData[k]; ok {
					return fmt.Sprintf("%v", v)
				}
			}
		}
	}
	for _, k := range keys {
		if v, ok := data[k]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func extractTotal(data map[string]interface{}) string {
	for _, wrapper := range []string{"order", "data"} {
		if wData, ok := data[wrapper].(map[string]interface{}); ok {
			if totals, ok := wData["order_totals"].([]interface{}); ok {
				for _, t := range totals {
					if tMap, ok := t.(map[string]interface{}); ok {
						if title, ok := tMap["title"].(string); ok && strings.ToLower(title) == "total" {
							if val, ok := tMap["value"]; ok {
								return fmt.Sprintf("%v", val)
							}
						}
					}
				}
			}
		}
	}
	return extractFieldString(data, []string{"total", "amount"})
}

func normalizeOrderPayload(payload []byte) []byte {
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return payload
	}

	normalized := map[string]interface{}{
		"order": map[string]interface{}{},
	}
	orderObj := normalized["order"].(map[string]interface{})

	if t := extractFieldString(raw, []string{"table_number", "table_name", "table", "table_id"}); t != "" {
		orderObj["table_number"] = t
	}
	if id := extractFieldString(raw, []string{"order_id", "id"}); id != "" {
		orderObj["id"] = id
	}
	if c := extractFieldString(raw, []string{"customer_name", "customer", "client"}); c != "" {
		orderObj["customer_name"] = c
	}
	if tot := extractTotal(raw); tot != "" {
		orderObj["total"] = tot
	}

	// Extraer estado general de la orden
	orderStatus := extractFieldString(raw, []string{"status_name", "status_label", "status", "state"})
	if orderStatus == "" {
		orderStatus = "Recibido"
	}

	// Extraer array de ítems/platillos
	var itemsList []interface{}
	for _, wrapper := range []string{"order", "data"} {
		if wMap, ok := raw[wrapper].(map[string]interface{}); ok {
			for _, k := range []string{"order_menus", "items", "menus", "menu_items"} {
				if items, ok := wMap[k].([]interface{}); ok {
					itemsList = items
					break
				}
			}
		}
	}
	if len(itemsList) == 0 {
		for _, k := range []string{"order_menus", "items", "menus", "menu_items"} {
			if items, ok := raw[k].([]interface{}); ok {
				itemsList = items
				break
			}
		}
	}

	var normalizedItems []NormalizedOrderItem
	var totalProgressSum int
	for _, itemVal := range itemsList {
		if itemMap, ok := itemVal.(map[string]interface{}); ok {
			var itemID, itemName, itemStatus string

			for _, k := range []string{"catalog_id", "menu_id", "order_menu_id", "id"} {
				if v, ok := itemMap[k]; ok {
					itemID = fmt.Sprintf("%v", v)
					break
				}
			}

			for _, k := range []string{"name", "menu_name", "title"} {
				if v, ok := itemMap[k]; ok {
					itemName = fmt.Sprintf("%v", v)
					break
				}
			}

			for _, k := range []string{"status_label", "status", "state"} {
				if v, ok := itemMap[k]; ok {
					itemStatus = fmt.Sprintf("%v", v)
					break
				}
			}

			if itemStatus == "" {
				itemStatus = orderStatus
			}

			// Extraer opciones/modificadores
			var itemOptions []string
			for _, optKey := range []string{"order_options", "options", "menu_options"} {
				if opts, ok := itemMap[optKey].([]interface{}); ok {
					for _, optVal := range opts {
						if optMap, ok := optVal.(map[string]interface{}); ok {
							for _, nameKey := range []string{"name", "value", "option_value_name"} {
								if nameVal, ok := optMap[nameKey]; ok {
									optStr := fmt.Sprintf("%v", nameVal)
									if optStr != "" {
										itemOptions = append(itemOptions, optStr)
										break
									}
								}
							}
						}
					}
					break
				}
			}

			progress := getProgressFromStatus(itemStatus)
			totalProgressSum += progress

			normalizedItems = append(normalizedItems, NormalizedOrderItem{
				ID:          itemID,
				Name:        itemName,
				StatusLabel: itemStatus,
				Options:     itemOptions,
			})
		}
	}

	orderObj["items"] = normalizedItems
	var progressPct int
	if len(normalizedItems) > 0 {
		progressPct = totalProgressSum / len(normalizedItems)
	} else {
		progressPct = getProgressFromStatus(orderStatus)
	}
	orderObj["order_progress_pct"] = progressPct

	result, err := json.Marshal(normalized)
	if err != nil {
		return payload
	}
	return result
}

func StartWebServer(port string) {
	events.Subscribe("tenant.*.pos.order", func(msg *nats.Msg) {
		ProcessCloudOrder(msg.Data)
	})

	http.HandleFunc("/ws/local", HandleLocalWS)

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pubHex, _ := db.GetSetting("public_key")

		status := map[string]interface{}{
			"status":            "online",
			"db_initialized":    db.DB != nil,
			"ed25519_publickey": pubHex,
		}

		for k, v := range events.GetStatus() {
			status[k] = v
		}

		json.NewEncoder(w).Encode(status)
	})

	http.HandleFunc("/api/auth/check", HandleAuthCheck)
	http.HandleFunc("/api/setup", HandleSetup)
	http.HandleFunc("/api/login", HandleLogin)
	http.HandleFunc("/api/logout", HandleLogout)
	http.HandleFunc("/api/settings/cloud", AuthMiddleware(HandleCloudSettings))
	http.HandleFunc("/api/settings/cloud/disconnect", AuthMiddleware(HandleCloudDisconnect))
	http.HandleFunc("/api/provision", AuthMiddleware(HandleProvisionUpload))
	http.HandleFunc("/api/settings/screensaver", AuthMiddleware(HandleScreensaverSettings))
	http.HandleFunc("/api/settings/screensaver/image", AuthMiddleware(HandleScreensaverImageUpload))
	http.HandleFunc("/api/device/screensaver", HandlePublicScreensaverConfig)

	http.HandleFunc("/api/users", AuthMiddleware(HandleUsers))
	http.HandleFunc("/api/users/", AuthMiddleware(HandleUsersByID))

	http.HandleFunc("/api/devices", AuthMiddleware(devices.HandleDevices))
	http.HandleFunc("/api/devices/create", AuthMiddleware(devices.HandleCreateDevice))
	http.HandleFunc("/api/devices/summary", AuthMiddleware(devices.HandleDeviceSummary))
	http.HandleFunc("/api/devices/provision_file", AuthMiddleware(devices.HandleGenerateProvisionFile))
	http.HandleFunc("/api/devices/provision-file", AuthMiddleware(devices.HandleGenerateProvisionFile))
	http.HandleFunc("/api/devices/approve", AuthMiddleware(devices.HandleApproveDevice))
	http.HandleFunc("/api/devices/block", AuthMiddleware(devices.HandleBlockDevice))
	http.HandleFunc("/api/alerts", AuthMiddleware(devices.HandleAlerts))

	http.HandleFunc("/api/settings/wifi", AuthMiddleware(devices.HandleWifiNetworks))

	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./data/uploads"))))
	http.Handle("/api/assets/", http.StripPrefix("/api/assets/", http.FileServer(http.Dir("./data/assets/processed"))))
	http.HandleFunc("/api/cdn/image", cdn.HandleDynamicCDNImage)
	http.Handle("/cdn/assets/", http.StripPrefix("/cdn/assets/", http.FileServer(http.Dir("./data/cdn/assets"))))
	http.Handle("/", http.FileServer(http.Dir("./web/dist")))

	log.Printf("Levantando servidor web local en el puerto %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error al arrancar servidor web: %v", err)
	}
}
