package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tablehub/cloud/internal/domain/branch"
	"github.com/tablehub/cloud/pkg/types"
)

type branchRepo struct {
	pool *pgxpool.Pool
}

func NewBranchRepository(pool *pgxpool.Pool) branch.Repository {
	return &branchRepo{pool: pool}
}

func (r *branchRepo) Create(ctx context.Context, b *branch.Branch) error {
	query := `
		INSERT INTO branches (id, organization_id, name, address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		uuid.UUID(b.ID), b.OrganizationID, b.Name, b.Address,
		b.CreatedAt, b.UpdatedAt,
	).Scan(&b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return branch.ErrDuplicateBranch
		}
		return fmt.Errorf("insert branch: %w", err)
	}
	return nil
}

func (r *branchRepo) GetByID(ctx context.Context, id branch.BranchID) (*branch.Branch, error) {
	query := `
		SELECT id, organization_id, name, address, created_at, updated_at
		FROM branches
		WHERE id = $1`

	b := &branch.Branch{}
	var bID uuid.UUID
	err := r.pool.QueryRow(ctx, query, uuid.UUID(id)).Scan(
		&bID, &b.OrganizationID, &b.Name, &b.Address, &b.CreatedAt, &b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("branch not found")
		}
		return nil, fmt.Errorf("get branch by id: %w", err)
	}

	b.ID = branch.BranchID(bID)
	return b, nil
}

func (r *branchRepo) ListByOrganization(ctx context.Context, orgID types.OrganizationID) ([]*branch.Branch, error) {
	query := `
		SELECT id, organization_id, name, address, created_at, updated_at
		FROM branches
		WHERE organization_id = $1
		ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list branches by org: %w", err)
	}
	defer rows.Close()

	var results []*branch.Branch
	for rows.Next() {
		b := &branch.Branch{}
		var bID uuid.UUID
		err := rows.Scan(
			&bID, &b.OrganizationID, &b.Name, &b.Address, &b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan branch: %w", err)
		}
		b.ID = branch.BranchID(bID)
		results = append(results, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate branches: %w", err)
	}
	return results, nil
}

func (r *branchRepo) Update(ctx context.Context, b *branch.Branch) error {
	query := `
		UPDATE branches
		SET name = $1, address = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query, b.Name, b.Address, uuid.UUID(b.ID)).Scan(&b.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update branch: %w", err)
	}
	return nil
}

func (r *branchRepo) Delete(ctx context.Context, id branch.BranchID) error {
	query := `DELETE FROM branches WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, uuid.UUID(id))
	if err != nil {
		return fmt.Errorf("delete branch: %w", err)
	}
	return nil
}

func (r *branchRepo) CountByOrganization(ctx context.Context, orgID types.OrganizationID) (int, error) {
	query := `
		SELECT COUNT(id)
		FROM branches
		WHERE organization_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count branches by org: %w", err)
	}
	return count, nil
}
