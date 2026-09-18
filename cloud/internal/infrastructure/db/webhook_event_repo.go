package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
	"github.com/tablehub/cloud/pkg/types"
)

// WebhookEventRepo implements domainwebhook.EventRepository using PostgreSQL.
type WebhookEventRepo struct {
	pool *pgxpool.Pool
}

// NewWebhookEventRepo creates a new WebhookEventRepo.
func NewWebhookEventRepo(pool *pgxpool.Pool) *WebhookEventRepo {
	return &WebhookEventRepo{
		pool: pool,
	}
}

// LogEvent inserts a new webhook event. It handles idempotency.
func (r *WebhookEventRepo) LogEvent(ctx context.Context, event *domainwebhook.WebhookEvent) error {
	query := `
		INSERT INTO webhook_events (
			organization_id, provider, event_type, payload, idempotency_key, status
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (organization_id, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
		RETURNING id, received_at`

	err := r.pool.QueryRow(ctx, query,
		event.OrganizationID,
		event.Provider,
		event.EventType,
		event.Payload,
		event.IdempotencyKey,
		event.Status,
	).Scan(&event.ID, &event.ReceivedAt)

	if err != nil {
		// If no rows were returned, it means a conflict occurred and DO NOTHING took effect.
		if errors.Is(err, pgx.ErrNoRows) {
			return domainwebhook.ErrDuplicateEvent
		}
		return fmt.Errorf("failed to insert webhook event: %w", err)
	}

	return nil
}

// UpdateEventStatus updates the status of an existing webhook event.
func (r *WebhookEventRepo) UpdateEventStatus(ctx context.Context, eventID uuid.UUID, status domainwebhook.EventStatus, retryCount int, hubID *string, errorDetail *string) error {
	query := `
		UPDATE webhook_events
		SET status = $1,
			retry_count = $2,
			hub_id = $3,
			error_detail = $4,
			delivered_at = CASE WHEN $1 = 'delivered' THEN NOW() ELSE delivered_at END
		WHERE id = $5`

	tag, err := r.pool.Exec(ctx, query, status, retryCount, hubID, errorDetail, eventID)
	if err != nil {
		return fmt.Errorf("failed to update webhook event status: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domainwebhook.ErrEventNotFound
	}

	return nil
}

// GetEventsByOrganization retrieves a list of webhook events for a given organization.
func (r *WebhookEventRepo) GetEventsByOrganization(ctx context.Context, organizationID types.OrganizationID, limit, offset int) ([]*domainwebhook.WebhookEvent, error) {
	query := `
		SELECT id, organization_id, provider, event_type, payload, idempotency_key, status, retry_count, hub_id, error_detail, received_at, delivered_at
		FROM webhook_events
		WHERE organization_id = $1
		ORDER BY received_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, organizationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query webhook events: %w", err)
	}
	defer rows.Close()

	var events []*domainwebhook.WebhookEvent
	for rows.Next() {
		var e domainwebhook.WebhookEvent
		err := rows.Scan(
			&e.ID,
			&e.OrganizationID,
			&e.Provider,
			&e.EventType,
			&e.Payload,
			&e.IdempotencyKey,
			&e.Status,
			&e.RetryCount,
			&e.HubID,
			&e.ErrorDetail,
			&e.ReceivedAt,
			&e.DeliveredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook event: %w", err)
		}
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return events, nil
}

// GetPendingRetries retrieves up to limit events ready for retry, locking them with SKIP LOCKED.
func (r *WebhookEventRepo) GetPendingRetries(ctx context.Context, limit int, now time.Time) ([]*domainwebhook.WebhookEvent, error) {
	query := `
		SELECT id, organization_id, provider, event_type, payload, idempotency_key, status, retry_count, hub_id, error_detail, received_at, delivered_at, ack_deadline, next_retry_at, expires_at
		FROM webhook_events
		WHERE status IN ('delivered', 'hub_offline')
		  AND next_retry_at IS NOT NULL
		  AND next_retry_at <= $1
		  AND (expires_at IS NULL OR expires_at > $1)
		ORDER BY next_retry_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED`

	rows, err := r.pool.Query(ctx, query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending retries: %w", err)
	}
	defer rows.Close()

	var events []*domainwebhook.WebhookEvent
	for rows.Next() {
		var e domainwebhook.WebhookEvent
		err := rows.Scan(
			&e.ID,
			&e.OrganizationID,
			&e.Provider,
			&e.EventType,
			&e.Payload,
			&e.IdempotencyKey,
			&e.Status,
			&e.RetryCount,
			&e.HubID,
			&e.ErrorDetail,
			&e.ReceivedAt,
			&e.DeliveredAt,
			&e.AckDeadline,
			&e.NextRetryAt,
			&e.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook event: %w", err)
		}
		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	if len(events) == 0 {
		return nil, domainwebhook.ErrNoPendingEvents
	}

	return events, nil
}

// AcknowledgeEvent clears the ack_deadline for a successfully acknowledged event.
func (r *WebhookEventRepo) AcknowledgeEvent(ctx context.Context, eventID uuid.UUID) error {
	query := `
		UPDATE webhook_events
		SET ack_deadline = NULL,
			next_retry_at = NULL
		WHERE id = $1 AND status = 'delivered'`

	tag, err := r.pool.Exec(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("failed to acknowledge webhook event: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domainwebhook.ErrEventNotFound
	}

	return nil
}

// FailPendingACKs marks all delivered events for a given hub_id as hub_offline and schedules a retry.
func (r *WebhookEventRepo) FailPendingACKs(ctx context.Context, hubID string) (int64, error) {
	query := `
		UPDATE webhook_events
		SET status = 'hub_offline',
			ack_deadline = NULL,
			next_retry_at = NOW() + INTERVAL '5 seconds',
			error_detail = COALESCE(error_detail || '; ', '') || 'hub disconnected before ack'
		WHERE hub_id = $1 AND status = 'delivered' AND ack_deadline IS NOT NULL`

	tag, err := r.pool.Exec(ctx, query, hubID)
	if err != nil {
		return 0, fmt.Errorf("failed to fail pending acks: %w", err)
	}

	return tag.RowsAffected(), nil
}
