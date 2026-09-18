package db

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/tablehub/cloud/pkg/types"
)

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) user.Repository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (id, provider_user_id, organization_id, role, onboarding_state, name, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	var dbOrgID interface{} = nil
	if u.OrganizationID != uuid.Nil {
		dbOrgID = u.OrganizationID
	}

	_, err := r.pool.Exec(ctx, query,
		u.ID,
		u.ProviderUserID,
		dbOrgID,
		u.Role,
		u.OnboardingState,
		u.Name,
		u.Email,
		u.CreatedAt,
		u.UpdatedAt,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return user.ErrDuplicateProviderUserID
	}

	return err
}

func (r *userRepo) GetByID(ctx context.Context, id types.UserID) (*user.User, error) {
	query := `
		SELECT id, provider_user_id, organization_id, role, onboarding_state, name, email, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	u := &user.User{}
	var orgID uuid.NullUUID
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.ProviderUserID,
		&orgID,
		&u.Role,
		&u.OnboardingState,
		&u.Name,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if orgID.Valid {
		u.OrganizationID = types.OrganizationID(orgID.UUID)
	}
	return u, nil
}

func (r *userRepo) GetByProviderUserID(ctx context.Context, providerUserID string) (*user.User, error) {
	query := `
		SELECT id, provider_user_id, organization_id, role, onboarding_state, name, email, created_at, updated_at
		FROM users
		WHERE provider_user_id = $1
	`
	u := &user.User{}
	var orgID uuid.NullUUID
	err := r.pool.QueryRow(ctx, query, providerUserID).Scan(
		&u.ID,
		&u.ProviderUserID,
		&orgID,
		&u.Role,
		&u.OnboardingState,
		&u.Name,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if orgID.Valid {
		u.OrganizationID = types.OrganizationID(orgID.UUID)
	}
	return u, nil
}

func (r *userRepo) GetByOrganizationID(ctx context.Context, orgID types.OrganizationID) ([]*user.User, error) {
	query := `
		SELECT id, provider_user_id, organization_id, role, onboarding_state, name, email, created_at, updated_at
		FROM users
		WHERE organization_id = $1
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u := &user.User{}
		var orgID uuid.NullUUID
		err := rows.Scan(
			&u.ID, &u.ProviderUserID, &orgID, &u.Role, &u.OnboardingState, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if orgID.Valid {
			u.OrganizationID = types.OrganizationID(orgID.UUID)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *userRepo) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET role = $1, organization_id = $2, onboarding_state = $3, name = $4, email = $5, updated_at = NOW()
		WHERE id = $6
	`
	var dbOrgID interface{} = nil
	if u.OrganizationID != uuid.Nil {
		dbOrgID = u.OrganizationID
	}
	cmdTag, err := r.pool.Exec(ctx, query, u.Role, dbOrgID, u.OnboardingState, u.Name, u.Email, u.ID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *userRepo) Delete(ctx context.Context, id types.UserID) error {
	query := `DELETE FROM users WHERE id = $1`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}
