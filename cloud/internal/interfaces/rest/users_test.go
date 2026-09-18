package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tablehub/cloud/internal/domain/organization"
	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/tablehub/cloud/pkg/types"
)

// MockIdentityProvider mocks the user.IdentityProvider interface.
type MockIdentityProvider struct {
	CreateAndInviteUserFunc func(ctx context.Context, email, orgID, role string) (string, error)
	AssignRoleFunc          func(ctx context.Context, userID, orgID, role string) error
	DeleteUserFunc          func(ctx context.Context, userID string) error
	DeleteCalled            bool
	DeleteUserID            string
}

func (m *MockIdentityProvider) CreateAndInviteUser(ctx context.Context, email, orgID, role string) (string, error) {
	if m.CreateAndInviteUserFunc != nil {
		return m.CreateAndInviteUserFunc(ctx, email, orgID, role)
	}
	return "mock-provider-id", nil
}

func (m *MockIdentityProvider) AssignRole(ctx context.Context, userID, orgID, role string) error {
	if m.AssignRoleFunc != nil {
		return m.AssignRoleFunc(ctx, userID, orgID, role)
	}
	return nil
}

func (m *MockIdentityProvider) DeleteUser(ctx context.Context, userID string) error {
	m.DeleteCalled = true
	m.DeleteUserID = userID
	if m.DeleteUserFunc != nil {
		return m.DeleteUserFunc(ctx, userID)
	}
	return nil
}

func (m *MockIdentityProvider) CreateOrganization(ctx context.Context, name string) (string, error) {
	return "mock-org-123", nil
}

func (m *MockIdentityProvider) DeleteOrganization(ctx context.Context, orgID string) error {
	return nil
}

// MockUserRepository mocks the user.Repository interface.
type MockUserRepository struct {
	CreateFunc              func(ctx context.Context, u *user.User) error
	GetByIDFunc             func(ctx context.Context, id types.UserID) (*user.User, error)
	GetByProviderUserIDFunc func(ctx context.Context, providerUserID string) (*user.User, error)
	UpdateFunc              func(ctx context.Context, u *user.User) error
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, u)
	}
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id types.UserID) (*user.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByProviderUserID(ctx context.Context, providerUserID string) (*user.User, error) {
	if m.GetByProviderUserIDFunc != nil {
		return m.GetByProviderUserIDFunc(ctx, providerUserID)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByOrganizationID(ctx context.Context, orgID types.OrganizationID) ([]*user.User, error) {
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, u)
	}
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id types.UserID) error {
	return nil
}

// MockOrganizationRepository mocks the organization.Repository interface.
type MockOrganizationRepository struct {
	GetByIDFunc func(ctx context.Context, id types.OrganizationID) (*organization.Organization, error)
}

func (m *MockOrganizationRepository) Create(ctx context.Context, r *organization.Organization) error {
	return nil
}

func (m *MockOrganizationRepository) GetByID(ctx context.Context, id types.OrganizationID) (*organization.Organization, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return &organization.Organization{
		ID:            id,
		ProviderOrgID: "provider-org-123",
		Slug:          "test-org",
		Name:          "Test Org",
	}, nil
}

func (m *MockOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	return nil, nil
}

func (m *MockOrganizationRepository) List(ctx context.Context, limit, offset int) ([]*organization.Organization, error) {
	return nil, nil
}

func (m *MockOrganizationRepository) Update(ctx context.Context, r *organization.Organization) error {
	return nil
}

func (m *MockOrganizationRepository) Delete(ctx context.Context, id types.OrganizationID) error {
	return nil
}

func TestInviteUser_Success(t *testing.T) {
	orgID := uuid.New()
	mockProvider := &MockIdentityProvider{
		CreateAndInviteUserFunc: func(ctx context.Context, email, orgIDParam, role string) (string, error) {
			assert.Equal(t, "provider-org-123", orgIDParam)
			return "provider-user-123", nil
		},
	}
	mockRepo := &MockUserRepository{
		CreateFunc: func(ctx context.Context, u *user.User) error {
			assert.Equal(t, "provider-user-123", u.ProviderUserID)
			assert.Equal(t, types.OrganizationID(orgID), u.OrganizationID)
			assert.Equal(t, user.RoleAdmin, u.Role)
			assert.Equal(t, "test@tablehub.com", u.Email)
			return nil
		},
	}
	mockOrgRepo := &MockOrganizationRepository{}

	handler := NewUsersHandler(mockProvider, mockRepo, mockOrgRepo)
	r := chi.NewRouter()
	r.Post("/orgs/{orgID}/users/invite", handler.InviteUser)

	reqBody := `{"email":"test@tablehub.com","role":"admin"}`
	req := httptest.NewRequest("POST", "/orgs/"+orgID.String()+"/users/invite", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp InviteUserResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "provider-user-123", resp.ProviderUserID)
	assert.Equal(t, "test@tablehub.com", resp.Email)
	assert.Equal(t, "admin", resp.Role)
}

func TestInviteUser_SagaRollbackOnLocalDBCreationFailure(t *testing.T) {
	orgID := uuid.New()
	mockProvider := &MockIdentityProvider{
		CreateAndInviteUserFunc: func(ctx context.Context, email, orgIDParam, role string) (string, error) {
			return "provider-user-789", nil
		},
	}
	mockRepo := &MockUserRepository{
		CreateFunc: func(ctx context.Context, u *user.User) error {
			return errors.New("db write failed")
		},
	}
	mockOrgRepo := &MockOrganizationRepository{}

	handler := NewUsersHandler(mockProvider, mockRepo, mockOrgRepo)
	r := chi.NewRouter()
	r.Post("/orgs/{orgID}/users/invite", handler.InviteUser)

	reqBody := `{"email":"fail-db@tablehub.com","role":"viewer"}`
	req := httptest.NewRequest("POST", "/orgs/"+orgID.String()+"/users/invite", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to save user locally")
	assert.True(t, mockProvider.DeleteCalled)
	assert.Equal(t, "provider-user-789", mockProvider.DeleteUserID)
}
