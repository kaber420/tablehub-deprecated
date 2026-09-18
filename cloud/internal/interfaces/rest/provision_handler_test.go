package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/tablehub/cloud/internal/application/tunnel"
	"github.com/tablehub/cloud/internal/domain/hub"
	"github.com/tablehub/cloud/internal/domain/user"
	"github.com/tablehub/cloud/pkg/types"
)

// MockHubRepository mocks hub.Repository
type MockHubRepository struct {
	RegisterFunc                    func(ctx context.Context, h *hub.Hub) error
	GetByIDFunc                     func(ctx context.Context, id types.HubID) (*hub.Hub, error)
	UpdateFunc                      func(ctx context.Context, h *hub.Hub) error
	UpdatePublicKeyAndBurnTokenFunc func(ctx context.Context, id types.HubID, publicKey string) error
}

func (m *MockHubRepository) Register(ctx context.Context, h *hub.Hub) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, h)
	}
	return nil
}

func (m *MockHubRepository) GetByID(ctx context.Context, id types.HubID) (*hub.Hub, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, hub.ErrHubNotFound
}

func (m *MockHubRepository) GetByOrganization(ctx context.Context, organizationID types.OrganizationID) ([]*hub.Hub, error) {
	return nil, nil
}

func (m *MockHubRepository) CountByOrganization(ctx context.Context, organizationID types.OrganizationID) (int, error) {
	return 0, nil
}

func (m *MockHubRepository) GetOnline(ctx context.Context) ([]*hub.Hub, error) {
	return nil, nil
}

func (m *MockHubRepository) UpdateStatus(ctx context.Context, id types.HubID, status hub.ConnectionStatus, lastSeen *time.Time) error {
	return nil
}

func (m *MockHubRepository) UpdateMetadata(ctx context.Context, id types.HubID, meta hub.Metadata) error {
	return nil
}

func (m *MockHubRepository) Update(ctx context.Context, h *hub.Hub) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, h)
	}
	return nil
}

func (m *MockHubRepository) UpdatePublicKeyAndBurnToken(ctx context.Context, id types.HubID, publicKey string) error {
	if m.UpdatePublicKeyAndBurnTokenFunc != nil {
		return m.UpdatePublicKeyAndBurnTokenFunc(ctx, id, publicKey)
	}
	return nil
}

func (m *MockHubRepository) Delete(ctx context.Context, id types.HubID) error {
	return nil
}

func TestProvisionHandler_HandleProvision(t *testing.T) {
	orgID := types.NewOrganizationID()
	mockUser := &user.User{
		ID:             types.NewUserID(),
		OrganizationID: orgID,
		Role:           user.RoleOwner,
	}

	ctx := context.WithValue(context.Background(), authContextKey, mockUser)

	var updatedHub *hub.Hub
	hubID := uuid.New().String()
	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return &hub.Hub{
				ID:               id,
				OrganizationID:   orgID,
				ConnectionStatus: hub.StatusOffline,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			updatedHub = h
			return nil
		},
	}

	provisionUC := tunnel.NewProvisionHubUseCase(mockRepo, "wss://cloud.tablehub.com/ws")
	handler := NewProvisionHandler(provisionUC, mockRepo)

	reqBody := `{"branch_id":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest("POST", "/provision", bytes.NewBufferString(reqBody)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParams(req, map[string]string{"id": hubID, "orgID": orgID.String()})
	rec := httptest.NewRecorder()

	handler.HandleProvision(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp tunnel.ProvisionResult
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)

	assert.NotEmpty(t, resp.HubID)
	assert.Equal(t, orgID.String(), resp.OrganizationID)
	assert.Equal(t, "wss://cloud.tablehub.com/ws", resp.CloudWSURL)
	assert.NotEmpty(t, resp.BootstrapToken)

	// Verify DB state
	assert.NotNil(t, updatedHub)
	assert.Equal(t, resp.HubID, updatedHub.ID)
	assert.Equal(t, resp.BootstrapToken, updatedHub.BootstrapToken)
	assert.Empty(t, updatedHub.PublicKey) // Empty key initially
}

func TestProvisionHandler_HandleActivate_Success(t *testing.T) {
	hubID := "test-hub-123"
	token := "secret-bootstrap-token"
	pubKey := "test-public-key"

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			if id == hubID {
				return &hub.Hub{
					ID:             hubID,
					BootstrapToken: token,
				}, nil
			}
			return nil, hub.ErrHubNotFound
		},
		UpdatePublicKeyAndBurnTokenFunc: func(ctx context.Context, id types.HubID, key string) error {
			assert.Equal(t, hubID, id)
			assert.Equal(t, pubKey, key)
			return nil
		},
	}

	provisionUC := tunnel.NewProvisionHubUseCase(mockRepo, "wss://cloud.tablehub.com/ws")
	handler := NewProvisionHandler(provisionUC, mockRepo)

	reqPayload := ActivateRequest{
		HubID:          hubID,
		BootstrapToken: token,
		PublicKey:      pubKey,
	}
	reqBytes, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest("POST", "/activate", bytes.NewBuffer(reqBytes))
	rec := httptest.NewRecorder()

	handler.HandleActivate(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
	assert.Equal(t, hubID, resp["hub_id"])
}

func TestProvisionHandler_HandleActivate_InvalidToken(t *testing.T) {
	hubID := "test-hub-123"
	token := "wrong-token"
	pubKey := "test-public-key"

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return &hub.Hub{
				ID:             hubID,
				BootstrapToken: "correct-token",
			}, nil
		},
	}

	provisionUC := tunnel.NewProvisionHubUseCase(mockRepo, "wss://cloud.tablehub.com/ws")
	handler := NewProvisionHandler(provisionUC, mockRepo)

	reqPayload := ActivateRequest{
		HubID:          hubID,
		BootstrapToken: token,
		PublicKey:      pubKey,
	}
	reqBytes, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest("POST", "/activate", bytes.NewBuffer(reqBytes))
	rec := httptest.NewRecorder()

	handler.HandleActivate(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
