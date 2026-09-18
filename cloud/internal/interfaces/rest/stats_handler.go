package rest

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/tablehub/cloud/internal/domain/branch"
	"github.com/tablehub/cloud/internal/domain/hub"
	"github.com/tablehub/cloud/pkg/types"
)

type StatsHandler struct {
	branchRepo branch.Repository
	hubRepo    hub.Repository
}

func NewStatsHandler(branchRepo branch.Repository, hubRepo hub.Repository) *StatsHandler {
	return &StatsHandler{branchRepo: branchRepo, hubRepo: hubRepo}
}

type OrgStatsResponse struct {
	BranchesCount int `json:"branches_count"`
	HubsCount     int `json:"hubs_count"`
}

func (h *StatsHandler) GetOrgStats(w http.ResponseWriter, r *http.Request) {
	orgIDStr := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid orgID"})
		return
	}

	orgTypeID := types.OrganizationID(orgID)

	branchesCount, err := h.branchRepo.CountByOrganization(r.Context(), orgTypeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to count branches"})
		return
	}

	hubsCount, err := h.hubRepo.CountByOrganization(r.Context(), orgTypeID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to count hubs"})
		return
	}

	resp := OrgStatsResponse{
		BranchesCount: branchesCount,
		HubsCount:     hubsCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
