package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/tablehub/cloud/internal/application/tunnel"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
)

type DispatchWebhookUseCase struct {
	registry   tunnel.ConnectionRegistry
	eventRepo  domainwebhook.EventRepository
	retryDelay time.Duration
}

func NewDispatchWebhookUseCase(
	registry tunnel.ConnectionRegistry,
	eventRepo domainwebhook.EventRepository,
) *DispatchWebhookUseCase {
	return &DispatchWebhookUseCase{
		registry:  registry,
		eventRepo: eventRepo,
	}
}

func (uc *DispatchWebhookUseCase) Execute(ctx context.Context, event *domainwebhook.WebhookEvent) error {
	sender, err := uc.registry.GetByOrganizationID(event.OrganizationID.String())
	if err != nil {
		uc.eventRepo.UpdateEventStatus(context.WithoutCancel(ctx), event.ID, domainwebhook.StatusHubOffline, 0, nil, nil)
		return fmt.Errorf("hub offline: %w", err)
	}

	hubID := sender.HubID()

	msg := tunnel.TunnelMessage{
		Event:     event.EventType,
		Timestamp: time.Now().UnixMilli(),
		Payload:   event.Payload,
	}

	const maxRetries = 2

	retryDelay := uc.retryDelay
	if retryDelay == 0 {
		retryDelay = 1 * time.Second
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = sender.Send(msg)
		if err == nil {
			uc.eventRepo.UpdateEventStatus(context.WithoutCancel(ctx), event.ID, domainwebhook.StatusDelivered, attempt, &hubID, nil)
			return nil
		}

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				errMsg := ctx.Err().Error()
				uc.eventRepo.UpdateEventStatus(context.WithoutCancel(ctx), event.ID, domainwebhook.StatusFailed, attempt, &hubID, &errMsg)
				return ctx.Err()
			case <-time.After(retryDelay):
				// retry
			}
		}
	}

	errMsg := err.Error()
	uc.eventRepo.UpdateEventStatus(context.WithoutCancel(ctx), event.ID, domainwebhook.StatusFailed, maxRetries, &hubID, &errMsg)
	return fmt.Errorf("failed to deliver webhook after retries: %w", err)
}
