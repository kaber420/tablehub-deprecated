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
	"github.com/stretchr/testify/require"

	"github.com/tablehub/cloud/internal/domain/hub"
	"github.com/tablehub/cloud/pkg/types"
)

func withChiURLParams(r *http.Request, params map[string]string) *http.Request {
	ctx := chi.NewRouteContext()
	for k, v := range params {
		ctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
}

func strPtr(s string) *string { return &s }

func TestHubHandler_UpdateHub_Success_AllFields(t *testing.T) {
	hubID := uuid.New().String()
	orgID := uuid.New()
	existingHub := &hub.Hub{
		ID:               hubID,
		OrganizationID:   orgID,
		BranchID:         nil,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			assert.Equal(t, hubID, id)
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			assert.NotNil(t, h.BranchID)
			assert.Equal(t, "HW-Model-X", h.Metadata.HardwareModel)
			assert.Equal(t, "POS-Pro", h.Metadata.POSProvider)
			return nil
		},
	}

	handler := NewHubHandler(mockRepo)

	branchID := uuid.New().String()
	reqBody := UpdateHubRequest{
		BranchID:      &branchID,
		HardwareModel: strPtr("HW-Model-X"),
		POSProvider:   strPtr("POS-Pro"),
	}
	body, _ := json.Marshal(reqBody)

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)

	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp hub.Hub
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, hubID, resp.ID)
}

func TestHubHandler_UpdateHub_Success_OnlyBranchID(t *testing.T) {
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		BranchID:         nil,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			assert.NotNil(t, h.BranchID)
			assert.Empty(t, h.Metadata.HardwareModel)
			assert.Empty(t, h.Metadata.POSProvider)
			return nil
		},
	}

	handler := NewHubHandler(mockRepo)
	branchID := uuid.New().String()
	reqBody := UpdateHubRequest{BranchID: &branchID}
	body, _ := json.Marshal(reqBody)

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHubHandler_UpdateHub_Success_BranchIDSetToNil(t *testing.T) {
	bid := uuid.New()
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		BranchID:         &bid,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			assert.Nil(t, h.BranchID)
			return nil
		},
	}

	handler := NewHubHandler(mockRepo)
	emptyBranch := ""
	reqBody := UpdateHubRequest{BranchID: &emptyBranch}
	body, _ := json.Marshal(reqBody)

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHubHandler_UpdateHub_Success_OnlyHardwareModel(t *testing.T) {
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			assert.Equal(t, "HW-2000", h.Metadata.HardwareModel)
			assert.Empty(t, h.Metadata.POSProvider)
			return nil
		},
	}

	handler := NewHubHandler(mockRepo)
	reqBody := UpdateHubRequest{HardwareModel: strPtr("HW-2000")}
	body, _ := json.Marshal(reqBody)

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHubHandler_UpdateHub_Success_OnlyPOSProvider(t *testing.T) {
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			assert.Empty(t, h.Metadata.HardwareModel)
			assert.Equal(t, "POS-System", h.Metadata.POSProvider)
			return nil
		},
	}

	handler := NewHubHandler(mockRepo)
	reqBody := UpdateHubRequest{POSProvider: strPtr("POS-System")}
	body, _ := json.Marshal(reqBody)

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHubHandler_UpdateHub_Success_NilPayload(t *testing.T) {
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			assert.Nil(t, h.BranchID)
			assert.Empty(t, h.Metadata.HardwareModel)
			assert.Empty(t, h.Metadata.POSProvider)
			return nil
		},
	}

	handler := NewHubHandler(mockRepo)
	reqBody := UpdateHubRequest{}
	body, _ := json.Marshal(reqBody)

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHubHandler_UpdateHub_Success_EmptyBody(t *testing.T) {
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			assert.Nil(t, h.BranchID)
			return nil
		},
	}

	handler := NewHubHandler(mockRepo)

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHubHandler_UpdateHub_Error_EmptyID(t *testing.T) {
	handler := NewHubHandler(&MockHubRepository{})

	req := httptest.NewRequest("PUT", "/hubs/", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req = withChiURLParams(req, map[string]string{"id": ""})
	rec := httptest.NewRecorder()

	handler.UpdateHub(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "hub id is required")
}

func TestHubHandler_UpdateHub_Error_InvalidJSON(t *testing.T) {
	hubID := uuid.New().String()
	handler := NewHubHandler(&MockHubRepository{})

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestHubHandler_UpdateHub_Error_HubNotFound(t *testing.T) {
	hubID := uuid.New().String()
	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return nil, hub.ErrHubNotFound
		},
	}

	handler := NewHubHandler(mockRepo)
	body, _ := json.Marshal(UpdateHubRequest{})

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "hub not found")
}

func TestHubHandler_UpdateHub_Error_GetByIDInternalError(t *testing.T) {
	hubID := uuid.New().String()
	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return nil, errors.New("db connection lost")
		},
	}

	handler := NewHubHandler(mockRepo)
	body, _ := json.Marshal(UpdateHubRequest{})

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "internal server error")
}

func TestHubHandler_UpdateHub_Error_InvalidBranchID(t *testing.T) {
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
	}

	handler := NewHubHandler(mockRepo)
	invalidBranch := "not-a-uuid"
	reqBody := UpdateHubRequest{BranchID: &invalidBranch}
	body, _ := json.Marshal(reqBody)

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid branch_id")
}

func TestHubHandler_UpdateHub_Error_UpdateFails(t *testing.T) {
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		BranchID:         nil,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			return errors.New("db unavailable")
		},
	}

	handler := NewHubHandler(mockRepo)
	body, _ := json.Marshal(UpdateHubRequest{HardwareModel: strPtr("HW-X")})

	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to update hub")
}

func TestHubHandler_UpdateHub_Success_ExtraFieldsIgnored(t *testing.T) {
	hubID := uuid.New().String()
	existingHub := &hub.Hub{
		ID:               hubID,
		ConnectionStatus: hub.StatusOffline,
	}

	mockRepo := &MockHubRepository{
		GetByIDFunc: func(ctx context.Context, id types.HubID) (*hub.Hub, error) {
			return existingHub, nil
		},
		UpdateFunc: func(ctx context.Context, h *hub.Hub) error {
			assert.Equal(t, "HW-Model", h.Metadata.HardwareModel)
			return nil
		},
	}

	handler := NewHubHandler(mockRepo)
	extraJSON := `{"hardware_model":"HW-Model","pos_provider":"POS-PRO","unknown_field":"ignored"}`
	r := chi.NewRouter()
	r.Put("/hubs/{id}", handler.UpdateHub)
	req := httptest.NewRequest("PUT", "/hubs/"+hubID, bytes.NewReader([]byte(extraJSON)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_UpdateHub_Registered(t *testing.T) {
	require.NotPanics(t, func() {
		r := chi.NewRouter()
		handler := NewHubHandler(&MockHubRepository{})
		r.Put("/hubs/{id}", handler.UpdateHub)
	})
}
