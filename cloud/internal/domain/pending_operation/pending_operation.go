package pendingop

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/tablehub/cloud/pkg/types"
)

// OpStatus defines the lifecycle states of a pending operation.
type OpStatus string

const (
	OpStatusPending   OpStatus = "pending"
	OpStatusRunning   OpStatus = "running"
	OpStatusCompleted OpStatus = "completed"
	OpStatusFailed    OpStatus = "failed"
)

// PendingOperation represents an asynchronous background task.
type PendingOperation struct {
	ID            uuid.UUID
	UserID        types.UserID
	OperationType string
	Payload       json.RawMessage
	Status        OpStatus
	Attempts      int
	MaxAttempts   int
	NextRetryAt   *time.Time
	LastError     *string
	Result        *json.RawMessage
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
