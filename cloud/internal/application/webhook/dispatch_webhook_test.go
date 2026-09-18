package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tablehub/cloud/internal/application/tunnel"
	domainwebhook "github.com/tablehub/cloud/internal/domain/webhook"
	"github.com/tablehub/cloud/pkg/types"
)

// --- mocks ---

type mockSender struct {
	hubID     string
	failCount int
	callCount int
}

func (s *mockSender) Send(_ tunnel.TunnelMessage) error {
	s.callCount++
	if s.callCount <= s.failCount {
		return errors.New("send failed")
	}
	return nil
}

func (s *mockSender) HubID() string { return s.hubID }

type mockConnectionRegistry struct {
	sender tunnel.Sender
	err    error
}

func (r *mockConnectionRegistry) Store(_, _ string, _ tunnel.Sender) {}
func (r *mockConnectionRegistry) Get(_ string) tunnel.Sender         { return nil }
func (r *mockConnectionRegistry) GetByOrganizationID(_ string) (tunnel.Sender, error) {
	return r.sender, r.err
}
func (r *mockConnectionRegistry) Delete(_ string)    {}
func (r *mockConnectionRegistry) Shutdown() []string { return nil }

type statusUpdateCall struct {
	eventID     uuid.UUID
	status      domainwebhook.EventStatus
	retryCount  int
	hubID       *string
	errorDetail *string
}

type mockEventRepository struct {
	mu          sync.Mutex
	updateCalls []statusUpdateCall
}

func (r *mockEventRepository) LogEvent(_ context.Context, _ *domainwebhook.WebhookEvent) error {
	return nil
}

func (r *mockEventRepository) UpdateEventStatus(_ context.Context, eventID uuid.UUID, status domainwebhook.EventStatus, retryCount int, hubID *string, errorDetail *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateCalls = append(r.updateCalls, statusUpdateCall{
		eventID:     eventID,
		status:      status,
		retryCount:  retryCount,
		hubID:       hubID,
		errorDetail: errorDetail,
	})
	return nil
}

func (r *mockEventRepository) GetEventsByOrganization(_ context.Context, _ types.OrganizationID, _, _ int) ([]*domainwebhook.WebhookEvent, error) {
	return nil, nil
}

func (r *mockEventRepository) GetPendingRetries(ctx context.Context, limit int, now time.Time) ([]*domainwebhook.WebhookEvent, error) {
	return nil, nil
}

func (r *mockEventRepository) AcknowledgeEvent(ctx context.Context, eventID uuid.UUID) error {
	return nil
}

func (r *mockEventRepository) FailPendingACKs(ctx context.Context, hubID string) (int64, error) {
	return 0, nil
}

// --- helpers ---

func newTestUseCase(sender tunnel.Sender, registryErr error, repo *mockEventRepository, retryDelay time.Duration) *DispatchWebhookUseCase {
	registry := &mockConnectionRegistry{sender: sender, err: registryErr}
	uc := NewDispatchWebhookUseCase(registry, repo)
	uc.retryDelay = retryDelay
	return uc
}

func baseEvent() *domainwebhook.WebhookEvent {
	return &domainwebhook.WebhookEvent{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		EventType:      "order.created",
		Payload:        json.RawMessage(`{"order_id":"abc"}`),
	}
}

const hubID = "hub-1"

// --- tests ---

func TestDispatch_Success_FirstAttempt(t *testing.T) {
	sender := &mockSender{hubID: hubID}
	repo := &mockEventRepository{}
	uc := newTestUseCase(sender, nil, repo, time.Millisecond)

	err := uc.Execute(context.Background(), baseEvent())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}
	c := repo.updateCalls[0]
	if c.status != domainwebhook.StatusDelivered {
		t.Fatalf("expected status %s, got %s", domainwebhook.StatusDelivered, c.status)
	}
	if c.retryCount != 0 {
		t.Fatalf("expected retryCount 0, got %d", c.retryCount)
	}
	if c.hubID == nil || *c.hubID != hubID {
		t.Fatalf("expected hubID %s, got %v", hubID, c.hubID)
	}
	if c.errorDetail != nil {
		t.Fatalf("expected nil errorDetail, got %v", *c.errorDetail)
	}
}

func TestDispatch_Success_AfterOneRetry(t *testing.T) {
	sender := &mockSender{hubID: hubID, failCount: 1}
	repo := &mockEventRepository{}
	uc := newTestUseCase(sender, nil, repo, time.Millisecond)

	err := uc.Execute(context.Background(), baseEvent())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}
	c := repo.updateCalls[0]
	if c.status != domainwebhook.StatusDelivered {
		t.Fatalf("expected status %s, got %s", domainwebhook.StatusDelivered, c.status)
	}
	if c.retryCount != 1 {
		t.Fatalf("expected retryCount 1, got %d", c.retryCount)
	}
}

func TestDispatch_Success_AfterTwoRetries(t *testing.T) {
	sender := &mockSender{hubID: hubID, failCount: 2}
	repo := &mockEventRepository{}
	uc := newTestUseCase(sender, nil, repo, time.Millisecond)

	err := uc.Execute(context.Background(), baseEvent())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}
	c := repo.updateCalls[0]
	if c.status != domainwebhook.StatusDelivered {
		t.Fatalf("expected status %s, got %s", domainwebhook.StatusDelivered, c.status)
	}
	if c.retryCount != 2 {
		t.Fatalf("expected retryCount 2, got %d", c.retryCount)
	}
}

func TestDispatch_AllRetriesExhausted(t *testing.T) {
	sender := &mockSender{hubID: hubID, failCount: 999}
	repo := &mockEventRepository{}
	uc := newTestUseCase(sender, nil, repo, time.Millisecond)

	err := uc.Execute(context.Background(), baseEvent())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}
	c := repo.updateCalls[0]
	if c.status != domainwebhook.StatusFailed {
		t.Fatalf("expected status %s, got %s", domainwebhook.StatusFailed, c.status)
	}
	if c.retryCount != 2 {
		t.Fatalf("expected retryCount 2 (maxRetries), got %d", c.retryCount)
	}
	if c.hubID == nil || *c.hubID != hubID {
		t.Fatalf("expected hubID %s, got %v", hubID, c.hubID)
	}
	if c.errorDetail == nil {
		t.Fatal("expected non-nil errorDetail")
	}
}

func TestDispatch_CancelDuringRetry(t *testing.T) {
	sender := &mockSender{hubID: hubID, failCount: 999}
	repo := &mockEventRepository{}
	uc := newTestUseCase(sender, nil, repo, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- uc.Execute(ctx, baseEvent())
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for dispatch to complete")
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}
	c := repo.updateCalls[0]
	if c.status != domainwebhook.StatusFailed {
		t.Fatalf("expected status %s, got %s", domainwebhook.StatusFailed, c.status)
	}
	if c.retryCount != 0 {
		t.Fatalf("expected retryCount 0, got %d", c.retryCount)
	}
	if c.hubID == nil || *c.hubID != hubID {
		t.Fatalf("expected hubID %s, got %v", hubID, c.hubID)
	}
	if c.errorDetail == nil {
		t.Fatal("expected non-nil errorDetail")
	}
}

func TestDispatch_PreCancelledContext(t *testing.T) {
	sender := &mockSender{hubID: hubID, failCount: 999}
	repo := &mockEventRepository{}
	uc := newTestUseCase(sender, nil, repo, time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := uc.Execute(ctx, baseEvent())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}
	c := repo.updateCalls[0]
	if c.status != domainwebhook.StatusFailed {
		t.Fatalf("expected status %s, got %s", domainwebhook.StatusFailed, c.status)
	}
	if c.retryCount != 0 {
		t.Fatalf("expected retryCount 0, got %d", c.retryCount)
	}
}

func TestDispatch_HubOffline(t *testing.T) {
	registryErr := errors.New("connection not found")
	repo := &mockEventRepository{}
	uc := newTestUseCase(nil, registryErr, repo, time.Millisecond)

	err := uc.Execute(context.Background(), baseEvent())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.updateCalls) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(repo.updateCalls))
	}
	c := repo.updateCalls[0]
	if c.status != domainwebhook.StatusHubOffline {
		t.Fatalf("expected status %s, got %s", domainwebhook.StatusHubOffline, c.status)
	}
	if c.retryCount != 0 {
		t.Fatalf("expected retryCount 0, got %d", c.retryCount)
	}
	if c.hubID != nil {
		t.Fatalf("expected nil hubID, got %v", *c.hubID)
	}
	if c.errorDetail != nil {
		t.Fatalf("expected nil errorDetail, got %v", *c.errorDetail)
	}
}
