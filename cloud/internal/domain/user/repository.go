package user

import (
	"context"

	"github.com/tablehub/cloud/pkg/types"
)

// Repository defines the persistence contract for users.
type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id types.UserID) (*User, error)
	GetByProviderUserID(ctx context.Context, providerUserID string) (*User, error)
	GetByOrganizationID(ctx context.Context, orgID types.OrganizationID) ([]*User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id types.UserID) error
}
