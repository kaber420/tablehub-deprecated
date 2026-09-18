package pendingop

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tablehub/cloud/pkg/types"
)

func TestOpStatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status OpStatus
		want   OpStatus
	}{
		{"pending", OpStatusPending, "pending"},
		{"running", OpStatusRunning, "running"},
		{"completed", OpStatusCompleted, "completed"},
		{"failed", OpStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status)
		})
	}
}

func TestPendingOperation_Defaults(t *testing.T) {
	op := &PendingOperation{}
	assert.Equal(t, uuid.UUID{}, op.ID)
	assert.Equal(t, uuid.Nil, uuid.UUID(op.UserID))
	assert.Empty(t, op.OperationType)
	assert.Nil(t, op.Payload)
	assert.Empty(t, op.Status)
	assert.Equal(t, 0, op.Attempts)
	assert.Equal(t, 0, op.MaxAttempts)
	assert.Nil(t, op.NextRetryAt)
	assert.Nil(t, op.LastError)
	assert.Nil(t, op.Result)
	assert.True(t, op.CreatedAt.IsZero())
	assert.True(t, op.UpdatedAt.IsZero())
}

func TestPendingOperation_FullInitialization(t *testing.T) {
	id := uuid.New()
	userID := types.NewUserID()
	now := time.Now().UTC()
	errMsg := "something went wrong"
	payload := json.RawMessage(`{"key":"value"}`)
	result := json.RawMessage(`{"status":"done"}`)
	nextRetry := now.Add(30 * time.Second)

	op := &PendingOperation{
		ID:            id,
		UserID:        userID,
		OperationType: "create_org",
		Payload:       payload,
		Status:        OpStatusPending,
		Attempts:      1,
		MaxAttempts:   3,
		NextRetryAt:   &nextRetry,
		LastError:     &errMsg,
		Result:        &result,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, id, op.ID)
	assert.Equal(t, userID, op.UserID)
	assert.Equal(t, "create_org", op.OperationType)
	assert.JSONEq(t, `{"key":"value"}`, string(op.Payload))
	assert.Equal(t, OpStatusPending, op.Status)
	assert.Equal(t, 1, op.Attempts)
	assert.Equal(t, 3, op.MaxAttempts)
	assert.Equal(t, nextRetry, *op.NextRetryAt)
	assert.Equal(t, errMsg, *op.LastError)
	assert.JSONEq(t, `{"status":"done"}`, string(*op.Result))
	assert.Equal(t, now, op.CreatedAt)
	assert.Equal(t, now, op.UpdatedAt)
}

func TestPendingOperation_StatusTransitions(t *testing.T) {
	op := &PendingOperation{Status: OpStatusPending}

	tests := []struct {
		name    string
		current OpStatus
		expect  string
	}{
		{"completed", OpStatusCompleted, "completed"},
		{"failed", OpStatusFailed, "failed"},
		{"running", OpStatusRunning, "running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op.Status = tt.current
			assert.Equal(t, string(tt.expect), string(op.Status))
		})
	}
}

func TestPendingOperation_RetryLogic(t *testing.T) {
	op := &PendingOperation{
		Attempts:    2,
		MaxAttempts: 5,
		NextRetryAt: nil,
	}

	assert.Equal(t, 2, op.Attempts)
	assert.Equal(t, 5, op.MaxAttempts)
	assert.True(t, op.Attempts < op.MaxAttempts)

	op.Attempts = 5
	assert.Equal(t, op.Attempts, op.MaxAttempts)
}

func TestPendingOperation_PayloadJSON(t *testing.T) {
	payload := json.RawMessage(`{"type":"test","data":123}`)
	op := &PendingOperation{
		Payload: payload,
	}

	raw, err := json.Marshal(op)
	require.NoError(t, err)

	var decoded PendingOperation
	err = json.Unmarshal(raw, &decoded)
	require.NoError(t, err)

	assert.JSONEq(t, string(payload), string(decoded.Payload))
}
