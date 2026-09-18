package webhook

import (
	"context"

	"github.com/tablehub/cloud/pkg/types"
)

// EventPublisher defines the interface to publish webhook events to a message broker (e.g. NATS JetStream)
type EventPublisher interface {
	PublishEvent(ctx context.Context, organizationID types.OrganizationID, event *WebhookEvent) error
}
