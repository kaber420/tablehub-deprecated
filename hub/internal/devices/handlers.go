package devices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tablehub/hub/internal/db"
	"github.com/tablehub/hub/internal/events"
)

type ApprovePayload struct {
	MAC           string  `json:"mac"`
	DeviceType    string  `json:"device_type"`
	TableNumber   *string `json:"table_number"`
	LocationName  *string `json:"location_name"`
	PublicKey     *string `json:"public_key"`
	WiFiNetworkID *int    `json:"wifi_network_id"`
	WiFiSSID      *string `json:"wifi_ssid"`
	WiFiPass      *string `json:"wifi_pass"`
}

type BlockPayload struct {
	MAC string `json:"mac"`
}

// HandleDevices maneja GET /api/devices (listar) y DELETE /api/devices?mac=... (eliminar).
func HandleDevices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		list, err := db.ListDevices(status)
		if err != nil {
			http.Error(w, "Error al listar dispositivos: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)

	case http.MethodDelete:
		mac := r.URL.Query().Get("mac")
		if mac == "" {
			http.Error(w, "Falta el parámetro 'mac'", http.StatusBadRequest)
			return
		}

		err := db.DeleteDevice(mac)
		if err != nil {
			http.Error(w, "Error al eliminar dispositivo: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "mac": mac})

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

type CreateDevicePayload struct {
	TableNumber  string `json:"table_number"`
	LocationName string `json:"location_name"`
	DeviceType   string `json:"device_type"`
}

// HandleCreateDevice maneja POST /api/devices/create (crear entidad de mesa/dispositivo).
func HandleCreateDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var payload CreateDevicePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.TableNumber == "" {
		http.Error(w, "Falta el campo obligatorio 'table_number'", http.StatusBadRequest)
		return
	}

	token, err := db.CreatePendingDevice(payload.TableNumber, payload.LocationName, payload.DeviceType)
	if err != nil {
		http.Error(w, "Error al crear mesa/dispositivo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":          "created",
		"table_number":    payload.TableNumber,
		"bootstrap_token": token,
	})
}

// HandleApproveDevice maneja POST /api/devices/approve (aprobar/editar asignación y sincronizar por MQTT).
func HandleApproveDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var payload ApprovePayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.MAC == "" {
		http.Error(w, "Falta el campo obligatorio 'mac'", http.StatusBadRequest)
		return
	}

	if payload.DeviceType == "" {
		payload.DeviceType = "table_pad" // Rol por defecto
	}

	err = db.ApproveDevice(payload.MAC, payload.DeviceType, payload.TableNumber, payload.LocationName, payload.PublicKey)
	if err != nil {
		http.Error(w, "Error al aprobar dispositivo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtener credenciales Wi-Fi si fueron seleccionadas o enviadas
	var wifiSSID, wifiPass string
	if payload.WiFiNetworkID != nil && *payload.WiFiNetworkID > 0 {
		net, err := db.GetWifiNetworkByID(*payload.WiFiNetworkID)
		if err == nil && net != nil {
			wifiSSID = net.SSID
			wifiPass = net.Password
		}
	} else if payload.WiFiSSID != nil && *payload.WiFiSSID != "" {
		wifiSSID = *payload.WiFiSSID
		if payload.WiFiPass != nil {
			wifiPass = *payload.WiFiPass
		}
	}

	// Publicar mensaje de reconfiguración remota por MQTT/NATS a la placa ESP32 en vivo
	cleanMAC := strings.ReplaceAll(strings.ReplaceAll(payload.MAC, ":", ""), "-", "")
	configMsg := map[string]interface{}{
		"type":        "device_config_update",
		"mac":         payload.MAC,
		"device_type": payload.DeviceType,
	}
	if payload.TableNumber != nil {
		configMsg["table_number"] = *payload.TableNumber
	}
	if payload.LocationName != nil {
		configMsg["location_name"] = *payload.LocationName
	}
	if wifiSSID != "" {
		configMsg["wifi_ssid"] = wifiSSID
		configMsg["wifi_pass"] = wifiPass
	}

	configBytes, _ := json.Marshal(configMsg)
	_ = events.Publish(fmt.Sprintf("tablehub.device.%s.config", cleanMAC), configBytes)
	_ = events.Publish("tablehub.device.all.config", configBytes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "approved",
		"mac":    payload.MAC,
		"synced": true,
	})
}

// HandleBlockDevice maneja POST /api/devices/block (bloquear).
func HandleBlockDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var payload BlockPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.MAC == "" {
		http.Error(w, "Falta el campo obligatorio 'mac'", http.StatusBadRequest)
		return
	}

	err = db.BlockDevice(payload.MAC)
	if err != nil {
		http.Error(w, "Error al bloquear dispositivo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "blocked", "mac": payload.MAC})
}

// HandleDeviceSummary maneja GET /api/devices/summary
func HandleDeviceSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	summary, err := db.GetDeviceSummary()
	if err != nil {
		http.Error(w, "Error al obtener resumen: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

type Alert struct {
	MAC      string `json:"mac"`
	Name     string `json:"name"`
	Device   db.Device `json:"device"`
	Message  string `json:"message"`
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Desc     string `json:"desc"`
}

// HandleAlerts maneja GET /api/alerts
func HandleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	devicesList, err := db.ListDevices("")
	if err != nil {
		http.Error(w, "Error al listar dispositivos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var list []Alert
	for _, d := range devicesList {
		name := ""
		if d.LocationName != nil && *d.LocationName != "" {
			name = *d.LocationName
		}
		if d.TableNumber != nil && *d.TableNumber != "" {
			name = "Mesa " + *d.TableNumber
		}
		if name == "" {
			name = d.DeviceType
		}

		if d.Status == "unprovisioned" {
			list = append(list, Alert{
				MAC:      d.MACAddress,
				Name:     "MAC: " + d.MACAddress,
				Device:   d,
				Message:  "Pendiente de aprobación",
				Type:     "unprovisioned",
				Severity: "info",
				Desc:     "Tipo: " + d.DeviceType,
			})
			continue
		}

		if d.Status == "active" {
			if d.BatteryLevel <= 20 {
				list = append(list, Alert{
					MAC:      d.MACAddress,
					Name:     name,
					Device:   d,
					Message:  "Batería críticamente baja",
					Type:     "battery_critical",
					Severity: "danger",
					Desc:     fmt.Sprintf("Nivel actual: %d%%", d.BatteryLevel),
				})
			} else if d.BatteryLevel <= 35 {
				list = append(list, Alert{
					MAC:      d.MACAddress,
					Name:     name,
					Device:   d,
					Message:  "Batería baja",
					Type:     "battery_warning",
					Severity: "warning",
					Desc:     fmt.Sprintf("Nivel actual: %d%%", d.BatteryLevel),
				})
			}

			if d.WiFiSignal != 0 && d.WiFiSignal <= -85 {
				list = append(list, Alert{
					MAC:      d.MACAddress,
					Name:     name,
					Device:   d,
					Message:  "Señal Wi-Fi inestable",
					Type:     "signal_weak",
					Severity: "warning",
					Desc:     fmt.Sprintf("Intensidad: %d dBm", d.WiFiSignal),
				})
			}
		}

		if d.Status == "offline" {
			list = append(list, Alert{
				MAC:      d.MACAddress,
				Name:     name,
				Device:   d,
				Message:  "Dispositivo desconectado",
				Type:     "offline",
				Severity: "danger",
				Desc:     "Dispositivo fuera de línea",
			})
		}
	}

	if list == nil {
		list = []Alert{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
