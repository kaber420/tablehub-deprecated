package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pendingop "github.com/tablehub/cloud/internal/domain/pending_operation"
	"github.com/tablehub/cloud/pkg/types"
)

// PendingOpRepo implements pendingop.Repository using PostgreSQL.
type PendingOpRepo struct {
	pool *pgxpool.Pool
}

// NewPendingOpRepo creates a new PendingOpRepo.
func NewPendingOpRepo(pool *pgxpool.Pool) *PendingOpRepo {
	return &PendingOpRepo{pool: pool}
}

// Create inserts a new pending operation into the database.
func (r *PendingOpRepo) Create(ctx context.Context, op *pendingop.PendingOperation) error {
	query := `
		INSERT INTO pending_operations (id, user_id, operation_type, payload, status, attempts, max_attempts, next_retry_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`
	_, err := r.pool.Exec(ctx, query,
		op.ID,
		types.UserID(op.UserID),
		op.OperationType,
		op.Payload,
		op.Status,
		op.Attempts,
		op.MaxAttempts,
		op.NextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert pending operation: %w", err)
	}
	return nil
}

// GetPendingOps fetches up to limit pending operations ready to be processed,
// locking them with SKIP LOCKED to prevent concurrent workers from double-processing.
func (r *PendingOpRepo) GetPendingOps(ctx context.Context, limit int, now time.Time) ([]*pendingop.PendingOperation, error) {
	query := `
		SELECT id, user_id, operation_type, payload, status, attempts, max_attempts, next_retry_at, last_error, result, created_at, updated_at
		FROM pending_operations
		WHERE status = 'pending'
		  AND (next_retry_at IS NULL OR next_retry_at <= $1)
		ORDER BY created_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.pool.Query(ctx, query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending operations: %w", err)
	}
	defer rows.Close()

	var ops []*pendingop.PendingOperation
	for rows.Next() {
		op := &pendingop.PendingOperation{}
		err := rows.Scan(
			&op.ID,
			&op.UserID,
			&op.OperationType,
			&op.Payload,
			&op.Status,
			&op.Attempts,
			&op.MaxAttempts,
			&op.NextRetryAt,
			&op.LastError,
			&op.Result,
			&op.CreatedAt,
			&op.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending operation: %w", err)
		}
		ops = append(ops, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return ops, nil
}

// UpdateStatus updates the status and metadata of an existing pending operation.
func (r *PendingOpRepo) UpdateStatus(ctx context.Context, opID uuid.UUID, status pendingop.OpStatus, attempts int, result *string, errDetail *string) error {
	query := `
		UPDATE pending_operations
		SET status = $1, attempts = $2, result = $3, last_error = $4, updated_at = NOW()
		WHERE id = $5
	`
	tag, err := r.pool.Exec(ctx, query, status, attempts, result, errDetail, opID)
	if err != nil {
		return fmt.Errorf("failed to update pending operation status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pendingop.ErrOpNotFound
	}
	return nil
}

// UpdateStatusWithRetry updates the status and schedules a retry for a failed operation.
func (r *PendingOpRepo) UpdateStatusWithRetry(ctx context.Context, opID uuid.UUID, status pendingop.OpStatus, attempts int, nextRetryAt *time.Time, errDetail *string) error {
	query := `
		UPDATE pending_operations
		SET status = $1, attempts = $2, next_retry_at = $3, last_error = $4, updated_at = NOW()
		WHERE id = $5
	`
	tag, err := r.pool.Exec(ctx, query, status, attempts, nextRetryAt, errDetail, opID)
	if err != nil {
		return fmt.Errorf("failed to update pending operation retry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pendingop.ErrOpNotFound
	}
	return nil
}

// GetByID retrieves a pending operation by its ID.
func (r *PendingOpRepo) GetByID(ctx context.Context, opID uuid.UUID) (*pendingop.PendingOperation, error) {
	query := `
		SELECT id, user_id, operation_type, payload, status, attempts, max_attempts, next_retry_at, last_error, result, created_at, updated_at
		FROM pending_operations
		WHERE id = $1
	`
	op := &pendingop.PendingOperation{}
	err := r.pool.QueryRow(ctx, query, opID).Scan(
		&op.ID,
		&op.UserID,
		&op.OperationType,
		&op.Payload,
		&op.Status,
		&op.Attempts,
		&op.MaxAttempts,
		&op.NextRetryAt,
		&op.LastError,
		&op.Result,
		&op.CreatedAt,
		&op.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pendingop.ErrOpNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pending operation: %w", err)
	}
	return op, nil
}

// GetActiveByUserID returns the active (pending or running) operation for a user, if any.
func (r *PendingOpRepo) GetActiveByUserID(ctx context.Context, userID types.UserID) (*pendingop.PendingOperation, error) {
	query := `
		SELECT id, user_id, operation_type, payload, status, attempts, max_attempts, next_retry_at, last_error, result, created_at, updated_at
		FROM pending_operations
		WHERE user_id = $1
		  AND status IN ('pending', 'running')
		LIMIT 1
	`
	op := &pendingop.PendingOperation{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&op.ID,
		&op.UserID,
		&op.OperationType,
		&op.Payload,
		&op.Status,
		&op.Attempts,
		&op.MaxAttempts,
		&op.NextRetryAt,
		&op.LastError,
		&op.Result,
		&op.CreatedAt,
		&op.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pendingop.ErrOpNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active operation for user: %w", err)
	}
	return op, nil
}
