package tunnel

import (
	"context"
	"fmt"
	"time"

	"github.com/tablehub/cloud/internal/domain/hub"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
	"github.com/tablehub/cloud/pkg/types"
)

// DisconnectHubUseCase handles the business logic when a hub disconnects.
type DisconnectHubUseCase struct {
	registry  ConnectionRegistry
	hubRepo   hub.Repository
	eventRepo domainwebhook.EventRepository
}

// NewDisconnectHubUseCase creates a new DisconnectHubUseCase with its dependencies.
func NewDisconnectHubUseCase(registry ConnectionRegistry, hubRepo hub.Repository, eventRepo domainwebhook.EventRepository) *DisconnectHubUseCase {
	return &DisconnectHubUseCase{
		registry:  registry,
		hubRepo:   hubRepo,
		eventRepo: eventRepo,
	}
}

// Execute removes the hub from the in-memory registry and updates the database
// status to "offline". This is called by the supervisor goroutine in the handler,
// never by the readPump's defer — preventing race conditions.
func (uc *DisconnectHubUseCase) Execute(ctx context.Context, hubID types.HubID) error {
	// 1. Remove from the in-memory registry.
	uc.registry.Delete(hubID)

	// 2. Mark all in-flight webhook events for this hub as failed.
	if _, err := uc.eventRepo.FailPendingACKs(ctx, string(hubID)); err != nil {
		return fmt.Errorf("disconnect hub %s: fail pending acks: %w", hubID, err)
	}

	// 3. Update the hub status to offline in the database.
	now := time.Now()
	if err := uc.hubRepo.UpdateStatus(ctx, hubID, hub.StatusOffline, &now); err != nil {
		return fmt.Errorf("disconnect hub %s: update status: %w", hubID, err)
	}

	return nil
}
