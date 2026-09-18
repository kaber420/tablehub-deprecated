package tunnel

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
)

// MessageHandler is the interface that the WebSocket layer uses to dispatch
// incoming messages from a hub to the application layer without coupling.
type MessageHandler interface {
	HandleMessage(hubID string, msg TunnelMessage)
}

// ForwardMessageUseCase routes messages between hubs using the connection registry.
// When a hub sends a message destined for a organization, this use case looks up
// the target hub's Sender and forwards the message.
// It also intercepts webhook.ack messages to acknowledge webhook delivery.
type ForwardMessageUseCase struct {
	registry  ConnectionRegistry
	eventRepo domainwebhook.EventRepository
}

// NewForwardMessageUseCase creates a new ForwardMessageUseCase.
func NewForwardMessageUseCase(registry ConnectionRegistry, eventRepo domainwebhook.EventRepository) *ForwardMessageUseCase {
	return &ForwardMessageUseCase{
		registry:  registry,
		eventRepo: eventRepo,
	}
}

// HandleMessage implements the MessageHandler interface.
// It processes incoming messages from hubs, routing them to the appropriate
// destination based on the message payload.
// It also intercepts webhook.ack messages to acknowledge webhook delivery asynchronously.
func (uc *ForwardMessageUseCase) HandleMessage(hubID string, msg TunnelMessage) {
	// Intercept webhook.ack messages.
	if msg.Event == "webhook.ack" {
		uc.handleWebhookACK(hubID, msg)
		return
	}

	// Extract target organization ID from the payload if it's a forward-type message.
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}

	targetOrganizationID, ok := payload["target_organization_id"].(string)
	if !ok || targetOrganizationID == "" {
		return
	}

	sender, err := uc.registry.GetByOrganizationID(targetOrganizationID)
	if err != nil {
		return
	}

	// Forward the message to the target hub.
	forwardMsg := TunnelMessage{
		Event:     msg.Event,
		Timestamp: msg.Timestamp,
		Payload:   payload,
	}
	if err := sender.Send(forwardMsg); err != nil {
		_ = fmt.Errorf("forward message from hub %s to organization %s: %w", hubID, targetOrganizationID, err)
	}
}

// handleWebhookACK processes a webhook.ack asynchronously to avoid blocking readPump.
func (uc *ForwardMessageUseCase) handleWebhookACK(hubID string, msg TunnelMessage) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}

	eventIDStr, ok := payload["event_id"].(string)
	if !ok || eventIDStr == "" {
		return
	}

	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := uc.eventRepo.AcknowledgeEvent(ctx, eventID); err != nil {
			_ = fmt.Errorf("acknowledge event %s from hub %s: %w", eventID, hubID, err)
		}
	}()
}
