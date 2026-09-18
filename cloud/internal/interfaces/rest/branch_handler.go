package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"errors"
	"github.com/tablehub/cloud/internal/domain/branch"
	"github.com/tablehub/cloud/pkg/types"
)

type BranchHandler struct {
	branchRepo branch.Repository
}

func NewBranchHandler(branchRepo branch.Repository) *BranchHandler {
	return &BranchHandler{branchRepo: branchRepo}
}

type CreateBranchRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (h *BranchHandler) CreateBranch(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid orgID"})
		return
	}

	var req CreateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Name) < 1 || len(req.Name) > 100 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "branch name must be between 1 and 100 characters"})
		return
	}

	b := &branch.Branch{
		ID:             branch.BranchID(uuid.New()),
		OrganizationID: types.OrganizationID(orgID),
		Name:           req.Name,
		Address:        req.Address,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.branchRepo.Create(r.Context(), b); err != nil {
		w.Header().Set("Content-Type", "application/json")
		if errors.Is(err, branch.ErrDuplicateBranch) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "branch already exists in this organization"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create branch"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

func (h *BranchHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid orgID"})
		return
	}

	results, err := h.branchRepo.ListByOrganization(r.Context(), types.OrganizationID(orgID))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list branches"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
