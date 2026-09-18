package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/tablehub/cloud/internal/domain/organization"
	pendingop "github.com/tablehub/cloud/internal/domain/pending_operation"
	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/tablehub/cloud/pkg/types"
)

const (
	OpTypeCreateOrg = "create_org"
	pollInterval    = 3 * time.Second
)

// CreateOrgPayload is the data stored in pending_operations.payload for a create_org operation.
type CreateOrgPayload struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	IdempotencyKey string `json:"idempotency_key"`
	// Stored after first attempt to recover from partial completion
	ProviderOrgID string `json:"provider_org_id,omitempty"`
	OrgDBID       string `json:"org_db_id,omitempty"`
}

// ReconciliationWorker processes pending onboarding operations sequentially.
type ReconciliationWorker struct {
	opRepo           pendingop.Repository
	userRepo         user.Repository
	orgRepo          organization.Repository
	identityProvider user.IdentityProvider
	logger           zerolog.Logger
}

// NewReconciliationWorker creates a new ReconciliationWorker.
func NewReconciliationWorker(
	opRepo pendingop.Repository,
	userRepo user.Repository,
	orgRepo organization.Repository,
	identityProvider user.IdentityProvider,
	logger zerolog.Logger,
) *ReconciliationWorker {
	return &ReconciliationWorker{
		opRepo:           opRepo,
		userRepo:         userRepo,
		orgRepo:          orgRepo,
		identityProvider: identityProvider,
		logger:           logger.With().Str("component", "reconciliation-worker").Logger(),
	}
}

// Start runs the worker loop until the context is cancelled.
func (w *ReconciliationWorker) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	w.logger.Info().Dur("poll_interval", pollInterval).Msg("Reconciliation worker started")

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("Reconciliation worker stopped")
			return
		case <-ticker.C:
			w.processPending(ctx)
		}
	}
}

func (w *ReconciliationWorker) processPending(ctx context.Context) {
	ops, err := w.opRepo.GetPendingOps(ctx, 5, time.Now())
	if err != nil {
		return // No pending ops or transient error — silently continue
	}

	for _, op := range ops {
		w.processOp(ctx, op)
	}
}

func (w *ReconciliationWorker) processOp(ctx context.Context, op *pendingop.PendingOperation) {
	logger := w.logger.With().Str("op_id", op.ID.String()).Str("type", op.OperationType).Str("user_id", op.UserID.String()).Logger()
	logger.Info().Int("attempt", op.Attempts+1).Msg("Processing pending operation")

	// Mark as running
	_ = w.opRepo.UpdateStatus(ctx, op.ID, pendingop.OpStatusRunning, op.Attempts, nil, nil)

	var execErr error
	switch op.OperationType {
	case OpTypeCreateOrg:
		execErr = w.executeCreateOrg(ctx, op, logger)
	default:
		errMsg := fmt.Sprintf("unknown operation type: %s", op.OperationType)
		logger.Error().Msg(errMsg)
		_ = w.opRepo.UpdateStatus(ctx, op.ID, pendingop.OpStatusFailed, op.Attempts+1, nil, &errMsg)
		return
	}

	if execErr == nil {
		logger.Info().Msg("Operation completed successfully")
		result := `{"status":"completed"}`
		_ = w.opRepo.UpdateStatus(ctx, op.ID, pendingop.OpStatusCompleted, op.Attempts+1, &result, nil)
		return
	}

	// Handle failure with exponential backoff retry
	newAttempts := op.Attempts + 1
	errMsg := execErr.Error()
	logger.Warn().Err(execErr).Int("attempt", newAttempts).Int("max", op.MaxAttempts).Msg("Operation failed")

	if newAttempts >= op.MaxAttempts {
		// Exhausted retries — mark user as org_failed
		logger.Error().Msg("Max attempts reached, marking operation as failed")
		_ = w.opRepo.UpdateStatus(ctx, op.ID, pendingop.OpStatusFailed, newAttempts, nil, &errMsg)
		w.markUserFailed(ctx, op.UserID, logger)
		return
	}

	// Schedule retry with exponential backoff: 10s, 30s, 90s...
	backoff := time.Duration(10*(1<<newAttempts)) * time.Second
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute
	}
	nextRetry := time.Now().Add(backoff)
	logger.Info().Dur("retry_in", backoff).Msg("Scheduling retry")
	_ = w.opRepo.UpdateStatusWithRetry(ctx, op.ID, pendingop.OpStatusPending, newAttempts, &nextRetry, &errMsg)
}

func (w *ReconciliationWorker) executeCreateOrg(ctx context.Context, op *pendingop.PendingOperation, logger zerolog.Logger) error {
	var payload CreateOrgPayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	opCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// Step 1: Create organization in Logto (idempotent — if it already exists from a previous attempt, we track the ID in the payload)
	providerOrgID := payload.ProviderOrgID
	if providerOrgID == "" {
		var err error
		providerOrgID, err = w.identityProvider.CreateOrganization(opCtx, payload.Name)
		if err != nil {
			return fmt.Errorf("create org in identity provider: %w", err)
		}
		logger.Info().Str("provider_org_id", providerOrgID).Msg("Organization created in identity provider")
	} else {
		logger.Info().Str("provider_org_id", providerOrgID).Msg("Reusing existing provider org from previous attempt")
	}

	// Step 2: Create organization in local DB (idempotent via orgDBID stored in payload)
	var orgDBID types.OrganizationID
	if payload.OrgDBID != "" {
		parsedID, err := uuid.Parse(payload.OrgDBID)
		if err == nil {
			orgDBID = types.OrganizationID(parsedID)
			logger.Info().Str("org_db_id", payload.OrgDBID).Msg("Reusing existing DB org from previous attempt")
		}
	}

	if orgDBID == uuid.Nil {
		newOrgID := uuid.New()
		org := &organization.Organization{
			ID:            types.OrganizationID(newOrgID),
			ProviderOrgID: providerOrgID,
			Slug:          payload.Slug,
			Name:          payload.Name,
			Plan:          organization.PlanFree,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := w.orgRepo.Create(opCtx, org); err != nil {
			if !errors.Is(err, organization.ErrDuplicateSlug) {
				// Rollback Logto org creation only if DB failed and we just created it
				if payload.ProviderOrgID == "" {
					_ = w.identityProvider.DeleteOrganization(opCtx, providerOrgID)
				}
				return fmt.Errorf("create org in database: %w", err)
			}
			// Slug conflict: the org already exists (from a previous retry) — try to fetch it
			logger.Warn().Msg("Slug conflict — org may already exist from a prior attempt, continuing")
		} else {
			orgDBID = types.OrganizationID(newOrgID)
			logger.Info().Str("org_db_id", newOrgID.String()).Msg("Organization created in database")
		}
	}

	// Step 3: Update user as owner — op.UserID is types.UserID (= uuid.UUID), look up directly by ID
	u, err := w.userRepo.GetByID(opCtx, op.UserID)
	if err != nil {
		return fmt.Errorf("get user for onboarding (id=%s): %w", op.UserID.String(), err)
	}

	u.OrganizationID = orgDBID
	u.Role = user.RoleOwner
	u.OnboardingState = user.StateActive
	if err := w.userRepo.Update(opCtx, u); err != nil {
		return fmt.Errorf("update user as owner: %w", err)
	}
	logger.Info().Str("user_id", u.ID.String()).Msg("User updated as organization owner")

	// Step 4: Assign role in Logto (best-effort — does not fail the saga)
	if err := w.identityProvider.AssignRole(opCtx, u.ProviderUserID, providerOrgID, string(user.RoleOwner)); err != nil {
		logger.Warn().Err(err).Msg("Failed to assign owner role in identity provider (best-effort, continuing)")
	} else {
		logger.Info().Msg("Owner role assigned in identity provider")
	}

	return nil
}

func (w *ReconciliationWorker) markUserFailed(ctx context.Context, userID types.UserID, logger zerolog.Logger) {
	userUUID := uuid.UUID(userID)
	u, err := w.userRepo.GetByID(ctx, types.UserID(userUUID))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to fetch user to mark as org_failed")
		return
	}
	u.OnboardingState = user.StateOrgFailed
	if err := w.userRepo.Update(ctx, u); err != nil {
		logger.Error().Err(err).Msg("Failed to update user state to org_failed")
	}
}
