package pendingop

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tablehub/cloud/pkg/types"
)

// Repository defines the contract for persisting pending operations.
type Repository interface {
	Create(ctx context.Context, op *PendingOperation) error
	GetByID(ctx context.Context, opID uuid.UUID) (*PendingOperation, error)
	GetPendingOps(ctx context.Context, limit int, now time.Time) ([]*PendingOperation, error)
	GetActiveByUserID(ctx context.Context, userID types.UserID) (*PendingOperation, error)
	UpdateStatus(ctx context.Context, opID uuid.UUID, status OpStatus, attempts int, result *string, errDetail *string) error
	UpdateStatusWithRetry(ctx context.Context, opID uuid.UUID, status OpStatus, attempts int, nextRetryAt *time.Time, errDetail *string) error
}
