package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tablehub/cloud/internal/domain/organization"
	"github.com/tablehub/cloud/pkg/types"
)

// OrganizationRepo implements organization.Repository using PostgreSQL.
type OrganizationRepo struct {
	pool *pgxpool.Pool
}

// NewOrganizationRepo creates a new PostgreSQL-backed organization repository.
func NewOrganizationRepo(pool *pgxpool.Pool) *OrganizationRepo {
	return &OrganizationRepo{pool: pool}
}

// Create inserts a new organization into the database.
func (r *OrganizationRepo) Create(ctx context.Context, rest *organization.Organization) error {
	settingsJSON, err := json.Marshal(rest.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	query := `
		INSERT INTO organizations (id, provider_org_id, slug, name, plan, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	err = r.pool.QueryRow(ctx, query,
		rest.ID, rest.ProviderOrgID, rest.Slug, rest.Name, string(rest.Plan), settingsJSON,
		rest.CreatedAt, rest.UpdatedAt,
	).Scan(&rest.CreatedAt, &rest.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "organizations_slug_unique" {
			return organization.ErrDuplicateSlug
		}
		return fmt.Errorf("insert organization: %w", err)
	}
	return nil
}

// GetByID retrieves a organization by its UUID.
func (r *OrganizationRepo) GetByID(ctx context.Context, id types.OrganizationID) (*organization.Organization, error) {
	return r.getOne(ctx, "SELECT id, provider_org_id, slug, name, plan, settings, created_at, updated_at FROM organizations WHERE id = $1", id)
}

// GetBySlug retrieves a organization by its unique slug.
func (r *OrganizationRepo) GetBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	return r.getOne(ctx, "SELECT id, provider_org_id, slug, name, plan, settings, created_at, updated_at FROM organizations WHERE slug = $1", slug)
}

// List retrieves a paginated list of organizations.
func (r *OrganizationRepo) List(ctx context.Context, limit, offset int) ([]*organization.Organization, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	query := `
		SELECT id, provider_org_id, slug, name, plan, settings, created_at, updated_at
		FROM organizations
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	var results []*organization.Organization
	for rows.Next() {
		rest, err := scanOrganization(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, rest)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organizations: %w", err)
	}
	return results, nil
}

// Update modifies an existing organization's mutable fields (name, plan, settings).
func (r *OrganizationRepo) Update(ctx context.Context, rest *organization.Organization) error {
	settingsJSON, err := json.Marshal(rest.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	query := `
		UPDATE organizations
		SET name = $1, plan = $2, settings = $3
		WHERE id = $4
		RETURNING updated_at`

	err = r.pool.QueryRow(ctx, query,
		rest.Name, string(rest.Plan), settingsJSON, rest.ID,
	).Scan(&rest.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return organization.ErrNotFound
		}
		return fmt.Errorf("update organization: %w", err)
	}
	return nil
}

// Delete removes a organization by ID. This cascades to hub_registry.
func (r *OrganizationRepo) Delete(ctx context.Context, id types.OrganizationID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM organizations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return organization.ErrNotFound
	}
	return nil
}

// getOne is a helper that executes a query expected to return exactly one organization.
func (r *OrganizationRepo) getOne(ctx context.Context, query string, args ...any) (*organization.Organization, error) {
	row := r.pool.QueryRow(ctx, query, args...)

	var rest organization.Organization
	var planStr string
	var settingsJSON []byte

	err := row.Scan(
		&rest.ID, &rest.ProviderOrgID, &rest.Slug, &rest.Name, &planStr, &settingsJSON,
		&rest.CreatedAt, &rest.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, organization.ErrNotFound
		}
		return nil, fmt.Errorf("scan organization: %w", err)
	}

	rest.Plan = organization.Plan(planStr)
	if err := json.Unmarshal(settingsJSON, &rest.Settings); err != nil {
		return nil, fmt.Errorf("unmarshal settings: %w", err)
	}
	return &rest, nil
}

// scanOrganization scans a single organization row from a pgx.Rows iterator.
func scanOrganization(rows pgx.Rows) (*organization.Organization, error) {
	var rest organization.Organization
	var planStr string
	var settingsJSON []byte

	err := rows.Scan(
		&rest.ID, &rest.ProviderOrgID, &rest.Slug, &rest.Name, &planStr, &settingsJSON,
		&rest.CreatedAt, &rest.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan organization row: %w", err)
	}

	rest.Plan = organization.Plan(planStr)
	if err := json.Unmarshal(settingsJSON, &rest.Settings); err != nil {
		return nil, fmt.Errorf("unmarshal settings: %w", err)
	}
	return &rest, nil
}
