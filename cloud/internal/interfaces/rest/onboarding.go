package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	appOnboarding "github.com/tablehub/cloud/internal/application/onboarding"
	"github.com/tablehub/cloud/internal/domain/organization"
	pendingop "github.com/tablehub/cloud/internal/domain/pending_operation"
	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/tablehub/cloud/pkg/types"
)

type OnboardingHandler struct {
	orgRepo          organization.Repository
	userRepo         user.Repository
	identityProvider user.IdentityProvider
	pendingOpRepo    pendingop.Repository
	logger           zerolog.Logger
}

func NewOnboardingHandler(
	orgRepo organization.Repository,
	userRepo user.Repository,
	identityProvider user.IdentityProvider,
	pendingOpRepo pendingop.Repository,
	logger zerolog.Logger,
) *OnboardingHandler {
	return &OnboardingHandler{
		orgRepo:          orgRepo,
		userRepo:         userRepo,
		identityProvider: identityProvider,
		pendingOpRepo:    pendingOpRepo,
		logger:           logger,
	}
}

type OnboardingRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type OnboardingResponse struct {
	OperationID string `json:"operation_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// HandleOnboarding enqueues an async org-creation operation and returns 202 Accepted immediately.
func (h *OnboardingHandler) HandleOnboarding(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With().Str("handler", "HandleOnboarding").Logger()
	logger.Info().Msg("Starting async onboarding")

	tokenInfo, ok := GetOIDCFromContext(r.Context())
	if !ok {
		logger.Warn().Msg("Missing OIDC token info")
		http.Error(w, `{"error":"unauthorized: OIDC token info missing"}`, http.StatusUnauthorized)
		return
	}
	logger.Info().Str("provider_user_id", tokenInfo.Sub).Str("email", tokenInfo.Email).Msg("User authenticated")

	var req OnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Slug == "" {
		http.Error(w, `{"error":"name and slug are required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Check if user already exists and their current state
	existingUser, err := h.userRepo.GetByProviderUserID(ctx, tokenInfo.Sub)

	if err == nil && existingUser != nil {
		// User already active — nothing to do
		if existingUser.OnboardingState == user.StateActive && existingUser.OrganizationID != uuid.Nil {
			logger.Warn().Msg("User already has an active organization")
			http.Error(w, `{"error":"user is already registered in an organization"}`, http.StatusConflict)
			return
		}
		// Check for existing active pending operation (anti double-click)
		activeOp, err := h.pendingOpRepo.GetActiveByUserID(ctx, existingUser.ID)
		if err == nil && activeOp != nil {
			logger.Warn().Str("op_id", activeOp.ID.String()).Msg("Active pending operation already exists for user")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(OnboardingResponse{
				OperationID: activeOp.ID.String(),
				Status:      string(activeOp.Status),
				Message:     "Organization creation is already in progress",
			})
			return
		}
	}

	// Create or update user with org_pending state
	var u *user.User
	if existingUser != nil {
		u = existingUser
		u.OnboardingState = user.StateOrgPending
		if err := h.userRepo.Update(ctx, u); err != nil {
			logger.Error().Err(err).Msg("Failed to update user state to org_pending")
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		logger.Info().Str("user_id", u.ID.String()).Msg("Updated existing user to org_pending")
	} else {
		userID := uuid.New()
		u = &user.User{
			ID:              types.UserID(userID),
			ProviderUserID:  tokenInfo.Sub,
			OnboardingState: user.StateOrgPending,
			Role:            user.RoleOwner,
			Name:            tokenInfo.Name,
			Email:           tokenInfo.Email,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := h.userRepo.Create(ctx, u); err != nil {
			if errors.Is(err, user.ErrDuplicateProviderUserID) {
				// Race condition: user was just created, retry the get
				u, _ = h.userRepo.GetByProviderUserID(ctx, tokenInfo.Sub)
			} else {
				logger.Error().Err(err).Msg("Failed to create user")
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				return
			}
		}
		logger.Info().Str("user_id", userID.String()).Msg("Created new user with org_pending state")
	}

	if u == nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Enqueue the create_org operation
	payload, err := json.Marshal(appOnboarding.CreateOrgPayload{
		Name:           req.Name,
		Slug:           req.Slug,
		IdempotencyKey: uuid.New().String(),
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	op := &pendingop.PendingOperation{
		ID:            uuid.New(),
		UserID:        types.UserID(u.ID),
		OperationType: appOnboarding.OpTypeCreateOrg,
		Payload:       json.RawMessage(payload),
		Status:        pendingop.OpStatusPending,
		Attempts:      0,
		MaxAttempts:   5,
	}

	if err := h.pendingOpRepo.Create(ctx, op); err != nil {
		logger.Error().Err(err).Msg("Failed to enqueue pending operation")
		http.Error(w, `{"error":"failed to start organization creation"}`, http.StatusInternalServerError)
		return
	}

	logger.Info().Str("op_id", op.ID.String()).Msg("Onboarding operation enqueued")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(OnboardingResponse{
		OperationID: op.ID.String(),
		Status:      "pending",
		Message:     "Organization creation started",
	})
}

// GetMe returns the current user's profile and onboarding state.
func (h *OnboardingHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	tokenInfo, ok := GetOIDCFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	u, err := h.userRepo.GetByProviderUserID(ctx, tokenInfo.Sub)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// User not in DB — they haven't started onboarding yet
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"registered":       false,
				"onboarding_state": string(user.StateNew),
				"email":            tokenInfo.Email,
				"name":             tokenInfo.Name,
			})
			return
		}
		h.logger.Error().Err(err).Msg("Database error in GetMe")
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	// User exists — return full state
	resp := map[string]interface{}{
		"registered":       u.OrganizationID != uuid.Nil && u.OnboardingState == user.StateActive,
		"onboarding_state": string(u.OnboardingState),
		"id":               u.ID.String(),
		"name":             u.Name,
		"email":            u.Email,
	}

	if u.OrganizationID != uuid.Nil {
		resp["organization_id"] = u.OrganizationID.String()
		resp["role"] = string(u.Role)

		// Fetch org name
		org, err := h.orgRepo.GetByID(ctx, u.OrganizationID)
		if err == nil && org != nil {
			resp["organization_name"] = org.Name
		}
	}

	// Include active pending operation info if applicable
	if u.OnboardingState == user.StateOrgPending || u.OnboardingState == user.StateOrgCreating {
		activeOp, err := h.pendingOpRepo.GetActiveByUserID(ctx, u.ID)
		if err == nil && activeOp != nil {
			resp["operation_id"] = activeOp.ID.String()
			resp["operation_status"] = string(activeOp.Status)
			resp["operation_attempts"] = activeOp.Attempts
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
