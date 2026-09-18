package rest

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/tablehub/cloud/internal/application/tunnel"
	"github.com/tablehub/cloud/internal/domain/hub"
)

// ProvisionHandler exposes endpoints for hub provisioning and activation.
type ProvisionHandler struct {
	provisionUC *tunnel.ProvisionHubUseCase
	hubRepo     hub.Repository
}

// NewProvisionHandler creates a new ProvisionHandler with its dependencies.
func NewProvisionHandler(provisionUC *tunnel.ProvisionHubUseCase, hubRepo hub.Repository) *ProvisionHandler {
	return &ProvisionHandler{
		provisionUC: provisionUC,
		hubRepo:     hubRepo,
	}
}

// HandleProvision generates a new hub bootstrap token and returns a .thub provisioning file.
func (h *ProvisionHandler) HandleProvision(w http.ResponseWriter, r *http.Request) {
	_, ok := GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	hubID := chi.URLParam(r, "id")
	if hubID == "" {
		http.Error(w, `{"error":"invalid hub id"}`, http.StatusBadRequest)
		return
	}

	orgID := chi.URLParam(r, "orgID")
	if orgID == "" {
		http.Error(w, `{"error":"invalid org id"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := h.provisionUC.Execute(ctx, orgID, hubID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Marshal the provisioning data as JSON for the .thub file content
	thubData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		http.Error(w, `{"error":"failed to generate provisioning file"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"tablehub.thub\"")
	w.WriteHeader(http.StatusOK)
	w.Write(thubData)
}

// ActivateRequest is the expected JSON body for the activation endpoint.
type ActivateRequest struct {
	HubID          string `json:"hub_id"`
	BootstrapToken string `json:"bootstrap_token"`
	PublicKey      string `json:"public_key"`
}

// HandleActivate registers the hub's public key using the bootstrap token.
func (h *ProvisionHandler) HandleActivate(w http.ResponseWriter, r *http.Request) {
	var req ActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.HubID == "" || req.BootstrapToken == "" || req.PublicKey == "" {
		http.Error(w, `{"error":"missing required fields"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 1. Fetch hub from database
	hubEntity, err := h.hubRepo.GetByID(ctx, req.HubID)
	if err != nil {
		if errors.Is(err, hub.ErrHubNotFound) {
			http.Error(w, `{"error":"hub not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	// 2. Validate token
	if hubEntity.BootstrapToken == "" || subtle.ConstantTimeCompare([]byte(hubEntity.BootstrapToken), []byte(req.BootstrapToken)) != 1 {
		http.Error(w, `{"error":"invalid bootstrap token"}`, http.StatusUnauthorized)
		return
	}

	// 3. Register public key and burn token
	err = h.hubRepo.UpdatePublicKeyAndBurnToken(ctx, req.HubID, req.PublicKey)
	if err != nil {
		http.Error(w, `{"error":"failed to activate hub: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"hub_id": req.HubID,
	})
}
