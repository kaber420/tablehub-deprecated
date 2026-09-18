package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tablehub/cloud/internal/domain/hub"
	"github.com/tablehub/cloud/pkg/types"
)

// HubHandler exposes REST endpoints for hub management.
type HubHandler struct {
	hubRepo hub.Repository
}

// NewHubHandler creates a new HubHandler.
func NewHubHandler(hubRepo hub.Repository) *HubHandler {
	return &HubHandler{hubRepo: hubRepo}
}

// HealthCheck responds with the service status.
func (h *HubHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "tablehub-cloud",
	})
}

// ListHubs returns all hubs belonging to the organization.
func (h *HubHandler) ListHubs(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid orgID"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	hubs, err := h.hubRepo.GetByOrganization(ctx, types.OrganizationID(orgID))
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hubs":  hubs,
		"total": len(hubs),
	})
}

// GetHub returns a single hub by ID.
func (h *HubHandler) GetHub(w http.ResponseWriter, r *http.Request) {
	hubID := chi.URLParam(r, "id")
	if hubID == "" {
		http.Error(w, `{"error":"hub id is required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	foundHub, err := h.hubRepo.GetByID(ctx, hubID)
	if err != nil {
		if err == hub.ErrHubNotFound {
			http.Error(w, `{"error":"hub not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(foundHub)
}

// CreateHub creates a new hub for the organization, applying initial configuration payload.
func (h *HubHandler) CreateHub(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid orgID"}`, http.StatusBadRequest)
		return
	}

	var req UpdateHubRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	var parsedBranchID *uuid.UUID
	if req.BranchID != nil && *req.BranchID != "" {
		bid, err := uuid.Parse(*req.BranchID)
		if err == nil {
			parsedBranchID = &bid
		}
	}

	// Prepare metadata from request
	var meta hub.Metadata
	if req.Name != nil {
		meta.Name = *req.Name
	}
	if req.HardwareModel != nil {
		meta.HardwareModel = *req.HardwareModel
	}
	if req.POSProvider != nil {
		meta.POSProvider = *req.POSProvider
	}

	hubID := uuid.New().String()
	now := time.Now()

	newHub := &hub.Hub{
		ID:               hubID,
		OrganizationID:   orgID,
		BranchID:         parsedBranchID,
		PublicKey:        "",
		ConnectionStatus: hub.StatusOffline,
		Metadata:         meta,
		BootstrapToken:   "",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.hubRepo.Register(ctx, newHub); err != nil {
		http.Error(w, `{"error":"failed to create hub"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newHub)
}

// UpdateHubRequest defines the payload for updating a hub.
type UpdateHubRequest struct {
	Name          *string `json:"name"`
	BranchID      *string `json:"branch_id"`
	HardwareModel *string `json:"hardware_model"`
	POSProvider   *string `json:"pos_provider"`
}

// UpdateHub updates the editable properties of an existing hub.
func (h *HubHandler) UpdateHub(w http.ResponseWriter, r *http.Request) {
	hubID := chi.URLParam(r, "id")
	if hubID == "" {
		http.Error(w, `{"error":"hub id is required"}`, http.StatusBadRequest)
		return
	}

	var req UpdateHubRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		http.Error(w, `{"error":"invalid request body `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 1. Get the existing hub
	existingHub, err := h.hubRepo.GetByID(ctx, hubID)
	if err != nil {
		if err == hub.ErrHubNotFound {
			http.Error(w, `{"error":"hub not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	// 2. Update fields
	if req.BranchID != nil {
		if *req.BranchID == "" {
			existingHub.BranchID = nil
		} else {
			bid, err := uuid.Parse(*req.BranchID)
			if err != nil {
				http.Error(w, `{"error":"invalid branch_id"}`, http.StatusBadRequest)
				return
			}
			existingHub.BranchID = &bid
		}
	}

	if req.HardwareModel != nil {
		existingHub.Metadata.HardwareModel = *req.HardwareModel
	}
	if req.POSProvider != nil {
		existingHub.Metadata.POSProvider = *req.POSProvider
	}
	if req.Name != nil {
		existingHub.Metadata.Name = *req.Name
	}

	// 3. Save
	if err := h.hubRepo.Update(ctx, existingHub); err != nil {
		http.Error(w, `{"error":"failed to update hub"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(existingHub)
}
