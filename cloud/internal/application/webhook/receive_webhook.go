package webhook

import (
	"context"
	"fmt"

	"github.com/tablehub/cloud/internal/domain/organization"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
	"github.com/tablehub/cloud/pkg/types"
)

type ReceiveWebhookUseCase struct {
	organizationRepo organization.Repository
	eventRepo        domainwebhook.EventRepository
	registry         ProviderRegistry
	publisher        domainwebhook.EventPublisher
}

func NewReceiveWebhookUseCase(
	organizationRepo organization.Repository,
	eventRepo domainwebhook.EventRepository,
	registry ProviderRegistry,
	publisher domainwebhook.EventPublisher,
) *ReceiveWebhookUseCase {
	return &ReceiveWebhookUseCase{
		organizationRepo: organizationRepo,
		eventRepo:        eventRepo,
		registry:         registry,
		publisher:        publisher,
	}
}

func (uc *ReceiveWebhookUseCase) Execute(ctx context.Context, providerName string, organizationID types.OrganizationID, r WebhookRequest) (*domainwebhook.WebhookEvent, error) {
	provider, err := uc.registry.Get(providerName)
	if err != nil {
		return nil, domainwebhook.ErrUnsupportedProvider
	}

	rest, err := uc.organizationRepo.GetByID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if err := provider.ValidateRequest(r, rest.Settings.WebhookToken); err != nil {
		return nil, domainwebhook.ErrInvalidToken
	}

	eventType, idempotencyKey, payload, err := provider.ParsePayload(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	event := &domainwebhook.WebhookEvent{
		OrganizationID: organizationID,
		Provider:       providerName,
		EventType:      eventType,
		Payload:        payload,
		IdempotencyKey: idempotencyKey,
		Status:         domainwebhook.StatusReceived,
	}

	if err := uc.eventRepo.LogEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to log event: %w", err)
	}

	if err := uc.publisher.PublishEvent(ctx, organizationID, event); err != nil {
		return event, fmt.Errorf("failed to publish event: %w", err)
	}

	uc.eventRepo.UpdateEventStatus(ctx, event.ID, domainwebhook.StatusDelivered, 1, nil, nil)

	return event, nil
}
