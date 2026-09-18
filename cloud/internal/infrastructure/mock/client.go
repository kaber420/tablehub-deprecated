package mock

import (
	"context"

	"github.com/google/uuid"
	"github.com/tablehub/cloud/internal/domain/user"
)

type mockIdentityProvider struct{}

// NewMockIdentityProvider creates an identity provider that performs no-op/mock operations.
func NewMockIdentityProvider() user.IdentityProvider {
	return &mockIdentityProvider{}
}

func (m *mockIdentityProvider) CreateAndInviteUser(ctx context.Context, email, orgID, role string) (string, error) {
	return "mock_user_" + uuid.NewString(), nil
}

func (m *mockIdentityProvider) AssignRole(ctx context.Context, userID, orgID, role string) error {
	return nil
}

func (m *mockIdentityProvider) DeleteUser(ctx context.Context, userID string) error {
	return nil
}

func (m *mockIdentityProvider) CreateOrganization(ctx context.Context, name string) (string, error) {
	return "mock_org_" + uuid.NewString(), nil
}

func (m *mockIdentityProvider) DeleteOrganization(ctx context.Context, orgID string) error {
	return nil
}
