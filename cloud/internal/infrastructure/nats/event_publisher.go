package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
	"github.com/tablehub/cloud/pkg/types"
)

type EventPublisher struct {
	js jetstream.JetStream
}

func NewEventPublisher(js jetstream.JetStream) *EventPublisher {
	return &EventPublisher{
		js: js,
	}
}

func (p *EventPublisher) PublishEvent(ctx context.Context, organizationID types.OrganizationID, event *domainwebhook.WebhookEvent) error {
	subject := fmt.Sprintf("tenant.%s.pos.order", organizationID.String())

	_, err := p.js.Publish(ctx, subject, []byte(event.Payload))
	if err != nil {
		return fmt.Errorf("failed to publish to jetstream: %w", err)
	}

	return nil
}
