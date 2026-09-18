package webhook

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tablehub/cloud/pkg/types"
)

// EventRepository defines the persistence contract for webhook events.
type EventRepository interface {
	LogEvent(ctx context.Context, event *WebhookEvent) error
	UpdateEventStatus(ctx context.Context, eventID uuid.UUID, status EventStatus, retryCount int, hubID *string, errorDetail *string) error
	GetEventsByOrganization(ctx context.Context, organizationID types.OrganizationID, limit, offset int) ([]*WebhookEvent, error)
	GetPendingRetries(ctx context.Context, limit int, now time.Time) ([]*WebhookEvent, error)
	AcknowledgeEvent(ctx context.Context, eventID uuid.UUID) error
	FailPendingACKs(ctx context.Context, hubID string) (int64, error)
}
