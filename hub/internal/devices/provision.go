package devices

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/tablehub/hub/internal/db"
	"github.com/tablehub/hub/internal/events"
	"golang.org/x/crypto/pbkdf2"
)

type OfflineProvisionRequest struct {
	TableID       string `json:"table_id"`
	WiFiSSID      string `json:"wifi_ssid"`
	WiFiPass      string `json:"wifi_pass"`
	PIN           string `json:"pin"`
	WifiNetworkID *int   `json:"wifi_network_id,omitempty"`
}

type offlinePayload struct {
	WiFiSSID       string `json:"wifi_ssid"`
	WiFiPass       string `json:"wifi_pass"`
	HubIP          string `json:"hub_ip"`
	MQTTPort       int    `json:"mqtt_port"`
	BootstrapToken string `json:"bootstrap_token"`
	TableID        string `json:"table_id"`
}

func HandleGenerateProvisionFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req OfflineProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.PIN == "" {
		http.Error(w, "Falta el campo obligatorio (pin)", http.StatusBadRequest)
		return
	}

	if req.WifiNetworkID != nil && *req.WifiNetworkID > 0 {
		network, err := db.GetWifiNetworkByID(*req.WifiNetworkID)
		if err != nil {
			http.Error(w, "Error al obtener red Wi-Fi: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if network == nil {
			http.Error(w, "Red Wi-Fi no encontrada", http.StatusBadRequest)
			return
		}
		req.WiFiSSID = network.SSID
		req.WiFiPass = network.Password
	}

	if req.TableID == "" || req.WiFiSSID == "" || req.WiFiPass == "" {
		http.Error(w, "Faltan campos obligatorios (table_id, wifi_ssid, wifi_pass)", http.StatusBadRequest)
		return
	}

	bootstrapToken, err := db.CreatePendingDevice(req.TableID, "", "table_pad")
	if err != nil {
		log.Printf("Error creando/obteniendo entidad de mesa: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	hubIP := getLocalIP()
	mqttPort := getMQTTPort()

	payload := offlinePayload{
		WiFiSSID:       req.WiFiSSID,
		WiFiPass:       req.WiFiPass,
		HubIP:          hubIP,
		MQTTPort:       mqttPort,
		BootstrapToken: bootstrapToken,
		TableID:        req.TableID,
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error serializando payload: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		log.Printf("Error generando salt: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	key := pbkdf2.Key([]byte(req.PIN), salt, 10000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("Error creando cifrador AES: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		log.Printf("Error generando nonce: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Printf("Error creando GCM: %v", err)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, 0, 8+12+len(ciphertext)+16)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="tablehub.enc"`)
	w.Write(out)
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func getMQTTPort() int {
	status := events.GetStatus()
	if mqttRaw, ok := status["mqtt_port"]; ok {
		if mqttFloat, ok := mqttRaw.(float64); ok {
			return int(mqttFloat)
		}
	}
	return 1883
}
