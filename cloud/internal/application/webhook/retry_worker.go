package webhook

import (
	"context"
	"sync"
	"time"

	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
)

type RetryWorker struct {
	eventRepo  domainwebhook.EventRepository
	publisher  domainwebhook.EventPublisher
}

func NewRetryWorker(eventRepo domainwebhook.EventRepository, publisher domainwebhook.EventPublisher) *RetryWorker {
	return &RetryWorker{
		eventRepo:  eventRepo,
		publisher:  publisher,
	}
}

func (w *RetryWorker) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processPending(ctx)
		}
	}
}

func (w *RetryWorker) processPending(ctx context.Context) {
	events, err := w.eventRepo.GetPendingRetries(ctx, 10, time.Now())
	if err != nil {
		return
	}
	for _, event := range events {
		go func(e *domainwebhook.WebhookEvent) {
			err := w.publisher.PublishEvent(ctx, e.OrganizationID, e)
			if err == nil {
				w.eventRepo.UpdateEventStatus(ctx, e.ID, domainwebhook.StatusDelivered, 1, nil, nil)
			}
		}(event)
	}
}
