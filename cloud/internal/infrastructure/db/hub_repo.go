package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/tablehub/cloud/internal/domain/hub"
	"github.com/tablehub/cloud/pkg/types"
)

// pgxPool is satisfied by *pgxpool.Pool and test mocks.
type pgxPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// HubRepo implements hub.Repository using PostgreSQL.
type HubRepo struct {
	pool pgxPool
}

// NewHubRepo creates a new PostgreSQL-backed hub repository.
func NewHubRepo(pool pgxPool) *HubRepo {
	return &HubRepo{pool: pool}
}

// Register inserts a new hub into the registry.
func (r *HubRepo) Register(ctx context.Context, h *hub.Hub) error {
	metadataJSON, err := json.Marshal(h.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO hub_registry (id, organization_id, branch_id, public_key, connection_status, last_seen, metadata, bootstrap_token, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at`

	var pubKey *string
	if h.PublicKey != "" {
		pubKey = &h.PublicKey
	}
	var token *string
	if h.BootstrapToken != "" {
		token = &h.BootstrapToken
	}

	err = r.pool.QueryRow(ctx, query,
		h.ID, h.OrganizationID, h.BranchID, pubKey, string(h.ConnectionStatus),
		h.LastSeen, metadataJSON, token, h.CreatedAt, h.UpdatedAt,
	).Scan(&h.CreatedAt, &h.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return hub.ErrDuplicateHub
		}
		return fmt.Errorf("register hub: %w", err)
	}
	return nil
}

// Update modifies an existing hub.
func (r *HubRepo) Update(ctx context.Context, h *hub.Hub) error {
	metadataJSON, err := json.Marshal(h.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		UPDATE hub_registry
		SET branch_id = $2, metadata = $3, bootstrap_token = $4, updated_at = $5
		WHERE id = $1`

	var token *string
	if h.BootstrapToken != "" {
		token = &h.BootstrapToken
	}

	h.UpdatedAt = time.Now()

	cmdTag, err := r.pool.Exec(ctx, query, h.ID, h.BranchID, metadataJSON, token, h.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update hub: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return hub.ErrHubNotFound
	}
	return nil
}

// GetByID retrieves a hub by its ID.
func (r *HubRepo) GetByID(ctx context.Context, id types.HubID) (*hub.Hub, error) {
	query := `
		SELECT id, organization_id, branch_id, public_key, connection_status, last_seen, metadata, bootstrap_token, created_at, updated_at
		FROM hub_registry
		WHERE id = $1`

	return r.scanOne(r.pool.QueryRow(ctx, query, id))
}

// GetByOrganization retrieves all hubs registered to a specific organization.
func (r *HubRepo) GetByOrganization(ctx context.Context, organizationID types.OrganizationID) ([]*hub.Hub, error) {
	query := `
		SELECT id, organization_id, branch_id, public_key, connection_status, last_seen, metadata, bootstrap_token, created_at, updated_at
		FROM hub_registry
		WHERE organization_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("get hubs by organization: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// CountByOrganization returns the total number of hubs registered to a specific organization.
func (r *HubRepo) CountByOrganization(ctx context.Context, organizationID types.OrganizationID) (int, error) {
	query := `
		SELECT COUNT(id)
		FROM hub_registry
		WHERE organization_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, organizationID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count hubs by organization: %w", err)
	}
	return count, nil
}

// GetOnline retrieves all hubs with connection_status = 'online'.
func (r *HubRepo) GetOnline(ctx context.Context) ([]*hub.Hub, error) {
	query := `
		SELECT id, organization_id, branch_id, public_key, connection_status, last_seen, metadata, bootstrap_token, created_at, updated_at
		FROM hub_registry
		WHERE connection_status = 'online'
		ORDER BY last_seen DESC NULLS LAST`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get online hubs: %w", err)
	}
	defer rows.Close()

	return r.scanMany(rows)
}

// UpdateStatus changes a hub's connection status and last_seen timestamp.
func (r *HubRepo) UpdateStatus(ctx context.Context, id types.HubID, status hub.ConnectionStatus, lastSeen *time.Time) error {
	query := `
		UPDATE hub_registry
		SET connection_status = $1, last_seen = $2
		WHERE id = $3`

	tag, err := r.pool.Exec(ctx, query, string(status), lastSeen, id)
	if err != nil {
		return fmt.Errorf("update hub status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return hub.ErrHubNotFound
	}
	return nil
}

// UpdateMetadata updates the metadata JSONB for a hub.
func (r *HubRepo) UpdateMetadata(ctx context.Context, id types.HubID, meta hub.Metadata) error {
	metadataJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		UPDATE hub_registry
		SET metadata = $1
		WHERE id = $2`

	tag, err := r.pool.Exec(ctx, query, metadataJSON, id)
	if err != nil {
		return fmt.Errorf("update hub metadata: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return hub.ErrHubNotFound
	}
	return nil
}

// UpdatePublicKeyAndBurnToken associates the public key to the hub and burns the bootstrap token.
func (r *HubRepo) UpdatePublicKeyAndBurnToken(ctx context.Context, id types.HubID, publicKey string) error {
	query := `
		UPDATE hub_registry
		SET public_key = $1, bootstrap_token = NULL, updated_at = NOW()
		WHERE id = $2`

	tag, err := r.pool.Exec(ctx, query, publicKey, id)
	if err != nil {
		return fmt.Errorf("update public key and burn token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return hub.ErrHubNotFound
	}
	return nil
}

// Delete removes a hub from the registry.
func (r *HubRepo) Delete(ctx context.Context, id types.HubID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM hub_registry WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete hub: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return hub.ErrHubNotFound
	}
	return nil
}

// scanOne scans a single hub row from a pgx.Row.
func (r *HubRepo) scanOne(row pgx.Row) (*hub.Hub, error) {
	var h hub.Hub
	var statusStr string
	var metadataJSON []byte
	var pubKey *string
	var token *string

	err := row.Scan(
		&h.ID, &h.OrganizationID, &h.BranchID, &pubKey, &statusStr,
		&h.LastSeen, &metadataJSON, &token, &h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, hub.ErrHubNotFound
		}
		return nil, fmt.Errorf("scan hub: %w", err)
	}

	h.ConnectionStatus = hub.ConnectionStatus(statusStr)
	if pubKey != nil {
		h.PublicKey = *pubKey
	}
	if token != nil {
		h.BootstrapToken = *token
	}
	if err := json.Unmarshal(metadataJSON, &h.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return &h, nil
}

// scanMany scans multiple hub rows from a pgx.Rows iterator.
func (r *HubRepo) scanMany(rows pgx.Rows) ([]*hub.Hub, error) {
	var results []*hub.Hub
	for rows.Next() {
		var h hub.Hub
		var statusStr string
		var metadataJSON []byte
		var pubKey *string
		var token *string

		err := rows.Scan(
			&h.ID, &h.OrganizationID, &h.BranchID, &pubKey, &statusStr,
			&h.LastSeen, &metadataJSON, &token, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan hub row: %w", err)
		}

		h.ConnectionStatus = hub.ConnectionStatus(statusStr)
		if pubKey != nil {
			h.PublicKey = *pubKey
		}
		if token != nil {
			h.BootstrapToken = *token
		}
		if err := json.Unmarshal(metadataJSON, &h.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
		results = append(results, &h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hubs: %w", err)
	}
	return results, nil
}
