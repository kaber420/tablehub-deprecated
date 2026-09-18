package web

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tablehub/hub/internal/crypto"
	"github.com/tablehub/hub/internal/db"
)

// CloudSettingsPayload representa la estructura JSON para la configuración de la nube y general.
type CloudSettingsPayload struct {
	CloudURL     string `json:"cloud_url"`
	RestaurantID string `json:"restaurant_id"`
	Language     string `json:"language"`
	PublicKey    string `json:"public_key"` // Solo lectura
	CloudEnabled bool   `json:"cloud_enabled"`
	IsConnected  bool   `json:"is_connected"` // Estado real activo del túnel WebSocket/NATS
}

// HandleCloudSettings maneja GET y POST en /api/settings/cloud
func HandleCloudSettings(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		cloudURL, _ := db.GetSetting("cloud_url")
		restaurantID, _ := db.GetSetting("restaurant_id")
		language, _ := db.GetSetting("language")
		pubKey, _ := db.GetSetting("public_key")
		cloudEnabledStr, _ := db.GetSetting("cloud_enabled")

		if language == "" {
			language = "es"
		}

		resp := CloudSettingsPayload{
			CloudURL:     cloudURL,
			RestaurantID: restaurantID,
			Language:     language,
			PublicKey:    pubKey,
			CloudEnabled: cloudEnabledStr == "true",
			IsConnected:  IsCloudConnected(),
		}


		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if r.Method == http.MethodPost {
		var req CloudSettingsPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		db.SaveSetting("cloud_url", req.CloudURL)
		db.SaveSetting("restaurant_id", req.RestaurantID)
		if req.Language != "" {
			db.SaveSetting("language", req.Language)
		}

		if req.CloudEnabled {
			db.SaveSetting("cloud_enabled", "true")
		} else {
			db.SaveSetting("cloud_enabled", "false")
		}

		RestartCloudConnection()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// InitCloudKeys verifica si las llaves Ed25519 existen, y de no ser así, las genera.
// También se encarga de generar y persistir el HubID (UUID v4) de forma autónoma.
// La clave privada se guarda en ./data/hub_cloud.key en lugar de SQLite.
// Si existe en SQLite (migración), la mueve al archivo y la elimina de la base de datos.
func InitCloudKeys() {
	hubIDStr, _ := db.GetSetting("hub_id")
	if hubIDStr == "" {
		newUUID := uuid.New().String()
		db.SaveSetting("hub_id", newUUID)
	} else {
		if _, err := uuid.Parse(hubIDStr); err != nil {
			newUUID := uuid.New().String()
			db.SaveSetting("hub_id", newUUID)
		}
	}

	keyFile := filepath.Join("data", "hub_cloud.key")

	privKeyHex, _ := db.GetSetting("private_key")
	if privKeyHex != "" {
		if err := os.MkdirAll(filepath.Dir(keyFile), 0700); err == nil {
			if err := os.WriteFile(keyFile, []byte(privKeyHex), 0600); err == nil {
				db.SaveSetting("private_key", "")
			}
		}
		return
	}

	if _, err := os.Stat(keyFile); err == nil {
		pubKeyHex, _ := db.GetSetting("public_key")
		if pubKeyHex == "" {
			privKeyHexBytes, err := os.ReadFile(keyFile)
			if err == nil {
				privKeyHex := strings.TrimSpace(string(privKeyHexBytes))
				privBytes, err := crypto.DecodeKeyFromHex(privKeyHex)
				if err == nil {
					privKey := ed25519.PrivateKey(privBytes)
					pubKey := privKey.Public().(ed25519.PublicKey)
					db.SaveSetting("public_key", hex.EncodeToString(pubKey))
				}
			}
		}
		return
	}

	pub, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		return
	}

	if err := os.MkdirAll(filepath.Dir(keyFile), 0700); err != nil {
		return
	}

	if err := os.WriteFile(keyFile, []byte(crypto.EncodeKeyToHex(priv)), 0600); err != nil {
		return
	}

	db.SaveSetting("public_key", crypto.EncodeKeyToHex(pub))
}

// ProvisionPayload representa la estructura JSON de un archivo de aprovisionamiento .thub.
type ProvisionPayload struct {
	HubID          string `json:"hub_id"`
	RestaurantID   string `json:"organization_id"`
	CloudWSURL     string `json:"cloud_url"`
	BootstrapToken string `json:"bootstrap_token"`
}

// HandleProvisionUpload procesa la carga del archivo .thub a través de POST /api/provision.
func HandleProvisionUpload(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 5MB)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Parse the .thub JSON
	var payload ProvisionPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		http.Error(w, "Invalid .thub file format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if payload.HubID == "" || payload.CloudWSURL == "" || payload.BootstrapToken == "" {
		http.Error(w, "Missing required fields in .thub file", http.StatusBadRequest)
		return
	}

	// Si el hub_id es diferente al anterior, o se había desconectado previamente, forzar recreación de llaves
	existingHubID, _ := db.GetSetting("hub_id")
	if existingHubID != "" && existingHubID != payload.HubID {
		log.Printf("Nuevo aprovisionamiento para Hub ID %s (anterior: %s). Limpiando llaves antiguas...", payload.HubID, existingHubID)
		db.SaveSetting("public_key", "")
		os.Remove(filepath.Join("data", "hub_cloud.key"))
	}

	// Asegurar generación local de llaves
	InitCloudKeys()

	// Obtener clave pública
	pubKeyHex, _ := db.GetSetting("public_key")
	if pubKeyHex == "" {
		http.Error(w, "Failed to load generated physical public key", http.StatusInternalServerError)
		return
	}

	// Guardar en base de datos local SQLite
	db.SaveSetting("hub_id", payload.HubID)
	db.SaveSetting("restaurant_id", payload.RestaurantID)
	db.SaveSetting("cloud_url", payload.CloudWSURL)
	db.SaveSetting("bootstrap_token", payload.BootstrapToken)

	// Derivar URL HTTP de activación
	activationURL := deriveHTTPURL(payload.CloudWSURL, "/v1/hubs/activate")

	// Petición POST a la Nube
	activationReqBody, err := json.Marshal(map[string]string{
		"hub_id":          payload.HubID,
		"bootstrap_token": payload.BootstrapToken,
		"public_key":      pubKeyHex,
	})
	if err != nil {
		http.Error(w, "Failed to marshal activation request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(activationURL, "application/json", bytes.NewBuffer(activationReqBody))
	if err != nil {
		log.Printf("Error requesting activation to Cloud at %s: %v", activationURL, err)
		http.Error(w, "Failed to contact Cloud for activation: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Cloud activation failed with status %d: %s", resp.StatusCode, string(respBody))
		http.Error(w, "Cloud activation failed: "+string(respBody), http.StatusBadRequest)
		return
	}

	// Activación exitosa
	db.SaveSetting("cloud_enabled", "true")
	db.SaveSetting("bootstrap_token", "") // Limpiar token

	log.Printf("Provisioning file applied successfully: hub_id=%s, restaurant_id=%s", payload.HubID, payload.RestaurantID)

	// Restart the cloud connection to apply changes
	RestartCloudConnection()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "hub_id": payload.HubID})
}

// deriveHTTPURL convierte una URL de WebSocket (ws/wss) en su URL HTTP equivalente.
func deriveHTTPURL(wsURL string, path string) string {
	u := wsURL
	if strings.HasPrefix(u, "wss://") {
		u = "https://" + strings.TrimPrefix(u, "wss://")
	} else if strings.HasPrefix(u, "ws://") {
		u = "http://" + strings.TrimPrefix(u, "ws://")
	}
	if idx := strings.LastIndex(u, "/ws"); idx != -1 {
		u = u[:idx]
	}
	return u + path
}

// HandleCloudDisconnect maneja la desconexión total y limpieza de credenciales SaaS/Cloud.
func HandleCloudDisconnect(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Limpiar ajustes en SQLite
	db.SaveSetting("cloud_url", "")
	db.SaveSetting("restaurant_id", "")
	db.SaveSetting("hub_id", "")
	db.SaveSetting("bootstrap_token", "")
	db.SaveSetting("public_key", "")
	db.SaveSetting("private_key", "")
	db.SaveSetting("cloud_enabled", "false")

	// 2. Eliminar archivo de clave privada en disco para forzar regeneración limpia en la próxima cuenta
	keyFile := filepath.Join("data", "hub_cloud.key")
	if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
		log.Printf("Advertencia al borrar %s durante la desconexión: %v", keyFile, err)
	}

	// 3. Detener la conexión activa inmediatamente
	RestartCloudConnection()

	log.Println("Hub desvinculado de la nube y credenciales eliminadas por completo.")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Disconnected and cleared cloud configuration"})
}


