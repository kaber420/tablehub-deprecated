package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tablehub/hub/internal/db"
)

// HandleUsers routes GET and POST requests for users
func HandleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := db.GetUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Role string `json:"role"`
			PIN  string `json:"pin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Role == "" || req.PIN == "" {
			http.Error(w, "Name, Role, and PIN are required", http.StatusBadRequest)
			return
		}

		if len(req.PIN) < 4 || len(req.PIN) > 6 {
			http.Error(w, "PIN must be between 4 and 6 characters", http.StatusBadRequest)
			return
		}

		id, err := db.CreateUser(req.Name, req.Role, req.PIN)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     id,
			"status": "ok",
		})
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// HandleUsersByID routes PUT and DELETE requests for a specific user
func HandleUsersByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/users/{id} or /api/users/{id}/pin
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	idStr := parts[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Check if this is a PIN reset: /api/users/{id}/pin
	if len(parts) == 2 && parts[1] == "pin" && r.Method == http.MethodPut {
		handleResetPIN(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name   string `json:"name"`
			Role   string `json:"role"`
			Active bool   `json:"active"`
			PIN    string `json:"pin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if req.PIN != "" && (len(req.PIN) < 4 || len(req.PIN) > 6) {
			http.Error(w, "PIN must be between 4 and 6 characters", http.StatusBadRequest)
			return
		}

		if err := db.UpdateUser(id, req.Name, req.Role, req.Active, req.PIN); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	case http.MethodDelete:
		if err := db.DeleteUser(id); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func handleResetPIN(w http.ResponseWriter, r *http.Request, id int) {
	var req struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if req.PIN == "" {
		http.Error(w, "PIN is required", http.StatusBadRequest)
		return
	}

	if len(req.PIN) < 4 || len(req.PIN) > 6 {
		http.Error(w, "PIN must be between 4 and 6 characters", http.StatusBadRequest)
		return
	}

	if err := db.UpdateUserPIN(id, req.PIN); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
