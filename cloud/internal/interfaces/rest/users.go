package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tablehub/cloud/internal/domain/organization"
	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/tablehub/cloud/pkg/types"
)

type UsersHandler struct {
	identityProvider user.IdentityProvider
	userRepo         user.Repository
	orgRepo          organization.Repository
}

func NewUsersHandler(identityProvider user.IdentityProvider, userRepo user.Repository, orgRepo organization.Repository) *UsersHandler {
	return &UsersHandler{
		identityProvider: identityProvider,
		userRepo:         userRepo,
		orgRepo:          orgRepo,
	}
}

type InviteUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type InviteUserResponse struct {
	ID             string `json:"id"`
	ProviderUserID string `json:"provider_user_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
}

func (h *UsersHandler) InviteUser(w http.ResponseWriter, r *http.Request) {
	orgIDParam := chi.URLParam(r, "orgID")
	parsedOrgID, err := uuid.Parse(orgIDParam)
	if err != nil {
		http.Error(w, `{"error":"invalid organization ID"}`, http.StatusBadRequest)
		return
	}

	var req InviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Role == "" {
		http.Error(w, `{"error":"email and role are required"}`, http.StatusBadRequest)
		return
	}

	// Validate roles allowed to be invited
	var targetRole user.Role
	switch req.Role {
	case string(user.RoleAdmin):
		targetRole = user.RoleAdmin
	case string(user.RoleViewer):
		targetRole = user.RoleViewer
	default:
		http.Error(w, `{"error":"invalid role, must be 'admin' or 'viewer'"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Fetch organization to get its ProviderOrgID
	org, err := h.orgRepo.GetByID(ctx, types.OrganizationID(parsedOrgID))
	if err != nil {
		http.Error(w, `{"error":"failed to find organization: `+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	// 1. Create/Invite in Identity Provider
	// This helper handles the user creation and project grant setup in the active IDP
	providerUserID, err := h.identityProvider.CreateAndInviteUser(ctx, req.Email, org.ProviderOrgID, string(targetRole))
	if err != nil {
		http.Error(w, `{"error":"failed to invite user in identity provider: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// 2. Create User Locally (DB)
	userID := uuid.New()
	u := &user.User{
		ID:             types.UserID(userID),
		ProviderUserID: providerUserID,
		OrganizationID: types.OrganizationID(parsedOrgID),
		Role:           targetRole,
		Name:           req.Email, // Default name to email before first login
		Email:          req.Email,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.userRepo.Create(ctx, u); err != nil {
		// Rollback IDP user creation if local DB write fails (Saga Compensation)
		_ = h.identityProvider.DeleteUser(ctx, providerUserID)
		http.Error(w, `{"error":"failed to save user locally: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(InviteUserResponse{
		ID:             u.ID.String(),
		ProviderUserID: u.ProviderUserID,
		Email:          u.Email,
		Role:           string(u.Role),
	})
}
