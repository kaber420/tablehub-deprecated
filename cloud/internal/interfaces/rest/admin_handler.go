package rest

import (
	"encoding/json"
	"net/http"

	"github.com/tablehub/cloud/internal/interfaces/ws"
)

// AdminHandler handles administrative REST API endpoints.
type AdminHandler struct {
	hubRegistry *ws.HubRegistry
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(hubRegistry *ws.HubRegistry) *AdminHandler {
	return &AdminHandler{
		hubRegistry: hubRegistry,
	}
}

// Stats returns basic statistics about the server's operation.
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	// For now, we only return the count of connected hubs.
	// In the future, this can be expanded with DB stats or memory metrics.
	connectedHubs := len(h.hubRegistry.GetAllHubIDs())

	response := map[string]interface{}{
		"connected_hubs": connectedHubs,
		"status":         "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Tunnels returns a list of currently connected hub tunnels.
func (h *AdminHandler) Tunnels(w http.ResponseWriter, r *http.Request) {
	hubIDs := h.hubRegistry.GetAllHubIDs()

	response := map[string]interface{}{
		"tunnels": hubIDs,
		"count":   len(hubIDs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
