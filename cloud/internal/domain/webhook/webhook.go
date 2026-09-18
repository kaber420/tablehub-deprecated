package webhook

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/tablehub/cloud/pkg/types"
)

// EventStatus represents the current state of a webhook event.
type EventStatus string

const (
	StatusReceived   EventStatus = "received"
	StatusDelivered  EventStatus = "delivered"
	StatusFailed     EventStatus = "failed"
	StatusHubOffline EventStatus = "hub_offline"
	StatusExpired    EventStatus = "expired"
)

// WebhookEvent represents a received webhook event and its delivery status.
type WebhookEvent struct {
	ID             uuid.UUID
	OrganizationID types.OrganizationID
	Provider       string
	EventType      string
	Payload        json.RawMessage
	IdempotencyKey *string
	Status         EventStatus
	RetryCount     int
	HubID          *string
	ErrorDetail    *string
	ReceivedAt     time.Time
	DeliveredAt    *time.Time
	AckDeadline    *time.Time
	NextRetryAt    *time.Time
	ExpiresAt      *time.Time
}
