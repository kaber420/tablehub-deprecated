package user

import "context"

// IdentityProvider defines the interface for communicating with an Identity Provider (Zitadel, Logto, etc.)
type IdentityProvider interface {
	CreateAndInviteUser(ctx context.Context, email, orgID, role string) (string, error)
	AssignRole(ctx context.Context, userID, orgID, role string) error
	DeleteUser(ctx context.Context, userID string) error
	CreateOrganization(ctx context.Context, name string) (string, error)
	DeleteOrganization(ctx context.Context, orgID string) error
}
