package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	pendingop "github.com/tablehub/cloud/internal/domain/pending_operation"
	"github.com/tablehub/cloud/pkg/auth"
	"github.com/tablehub/cloud/pkg/types"
)

// MockPendingOpRepo is a mock of pendingop.Repository for testing.
type MockPendingOpRepo struct{}

func (m *MockPendingOpRepo) Create(ctx context.Context, op *pendingop.PendingOperation) error {
	return nil
}
func (m *MockPendingOpRepo) GetByID(ctx context.Context, opID uuid.UUID) (*pendingop.PendingOperation, error) {
	return nil, pendingop.ErrOpNotFound
}
func (m *MockPendingOpRepo) GetPendingOps(ctx context.Context, limit int, now time.Time) ([]*pendingop.PendingOperation, error) {
	return nil, nil
}
func (m *MockPendingOpRepo) GetActiveByUserID(ctx context.Context, userID types.UserID) (*pendingop.PendingOperation, error) {
	return nil, pendingop.ErrOpNotFound
}
func (m *MockPendingOpRepo) UpdateStatus(ctx context.Context, opID uuid.UUID, status pendingop.OpStatus, attempts int, result *string, errDetail *string) error {
	return nil
}
func (m *MockPendingOpRepo) UpdateStatusWithRetry(ctx context.Context, opID uuid.UUID, status pendingop.OpStatus, attempts int, nextRetryAt *time.Time, errDetail *string) error {
	return nil
}

func TestOnboarding_Success(t *testing.T) {
	mockProvider := &MockIdentityProvider{}
	mockOrgRepo := &MockOrganizationRepository{}
	mockUserRepo := &MockUserRepository{}
	mockPendingOpRepo := &MockPendingOpRepo{}

	logger := zerolog.Nop()

	tokenInfo := &auth.TokenInfo{
		Sub:   "provider-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}
	ctx := context.WithValue(context.Background(), oidcContextKey, tokenInfo)

	handler := NewOnboardingHandler(mockOrgRepo, mockUserRepo, mockProvider, mockPendingOpRepo, logger)

	reqBody := `{"name":"Acme Corp","slug":"acme"}`
	req := httptest.NewRequest("POST", "/onboard", bytes.NewBufferString(reqBody)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Post("/onboard", handler.HandleOnboarding)
	r.ServeHTTP(rec, req)

	// New async flow: returns 202 Accepted with operation_id
	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp OnboardingResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "pending", resp.Status)
	assert.NotEmpty(t, resp.OperationID)
}
