package devices

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/tablehub/hub/internal/db"
)

type SaveWifiPayload struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

func HandleWifiNetworks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := db.ListWifiNetworks()
		if err != nil {
			http.Error(w, "Error al listar redes Wi-Fi: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		var payload SaveWifiPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Cuerpo de solicitud inválido: "+err.Error(), http.StatusBadRequest)
			return
		}
		if payload.SSID == "" || payload.Password == "" {
			http.Error(w, "Faltan campos obligatorios (ssid, password)", http.StatusBadRequest)
			return
		}
		if err := db.SaveWifiNetwork(payload.SSID, payload.Password); err != nil {
			http.Error(w, "Error al guardar red Wi-Fi: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "saved", "ssid": payload.SSID})

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "Falta el parámetro 'id'", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "El parámetro 'id' debe ser un número", http.StatusBadRequest)
			return
		}
		if err := db.DeleteWifiNetwork(id); err != nil {
			http.Error(w, "Error al eliminar red Wi-Fi: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}
