package tunnel

import (
	"context"
	"fmt"
	"time"

	"github.com/tablehub/cloud/internal/domain/hub"
	"github.com/tablehub/cloud/pkg/types"
)

// ConnectHubUseCase handles the business logic when a hub establishes a connection.
type ConnectHubUseCase struct {
	registry ConnectionRegistry
	hubRepo  hub.Repository
}

// NewConnectHubUseCase creates a new ConnectHubUseCase with its dependencies.
func NewConnectHubUseCase(registry ConnectionRegistry, hubRepo hub.Repository) *ConnectHubUseCase {
	return &ConnectHubUseCase{
		registry: registry,
		hubRepo:  hubRepo,
	}
}

// Execute registers the hub's Sender in the in-memory registry and updates
// the database status to "online" with the current timestamp.
func (uc *ConnectHubUseCase) Execute(ctx context.Context, hubID types.HubID, organizationID string, sender Sender) error {
	// 1. Register the sender in the in-memory connection registry.
	uc.registry.Store(hubID, organizationID, sender)

	// 2. Update the hub status to online in the database.
	now := time.Now()
	if err := uc.hubRepo.UpdateStatus(ctx, hubID, hub.StatusOnline, &now); err != nil {
		// If DB update fails, remove from registry to maintain consistency.
		uc.registry.Delete(hubID)
		return fmt.Errorf("connect hub %s: update status: %w", hubID, err)
	}

	return nil
}
