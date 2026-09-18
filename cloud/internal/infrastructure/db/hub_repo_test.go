package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tablehub/cloud/internal/domain/hub"
)

func TestHubRepo_Update_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewHubRepo(mock)

	bid := uuid.New()
	h := &hub.Hub{
		ID:             uuid.New().String(),
		BranchID:       &bid,
		Metadata:       hub.Metadata{HardwareModel: "HW-X", POSProvider: "POS-Y"},
		BootstrapToken: "token-123",
		UpdatedAt:      time.Now().Add(-1 * time.Hour),
	}

	mock.ExpectExec("UPDATE hub_registry").
		WithArgs(h.ID, h.BranchID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), h)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.WithinDuration(t, time.Now(), h.UpdatedAt, 2*time.Second)
}

func TestHubRepo_Update_Success_BranchIDNil(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewHubRepo(mock)

	h := &hub.Hub{
		ID:             uuid.New().String(),
		BranchID:       nil,
		Metadata:       hub.Metadata{FirmwareVersion: "v1.0"},
		BootstrapToken: "",
		UpdatedAt:      time.Now().Add(-1 * time.Hour),
	}

	mock.ExpectExec("UPDATE hub_registry").
		WithArgs(h.ID, (*uuid.UUID)(nil), pgxmock.AnyArg(), (*string)(nil), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), h)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHubRepo_Update_Success_EmptyMetadata(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewHubRepo(mock)

	bid := uuid.New()
	h := &hub.Hub{
		ID:             uuid.New().String(),
		BranchID:       &bid,
		Metadata:       hub.Metadata{},
		BootstrapToken: "",
		UpdatedAt:      time.Now(),
	}

	mock.ExpectExec("UPDATE hub_registry").
		WithArgs(h.ID, h.BranchID, pgxmock.AnyArg(), (*string)(nil), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), h)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHubRepo_Update_Success_WithBootstrapToken(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewHubRepo(mock)

	bid := uuid.New()
	token := "my-bootstrap-token"
	h := &hub.Hub{
		ID:             uuid.New().String(),
		BranchID:       &bid,
		Metadata:       hub.Metadata{POSProvider: "POS-A"},
		BootstrapToken: token,
		UpdatedAt:      time.Now(),
	}

	mock.ExpectExec("UPDATE hub_registry").
		WithArgs(h.ID, h.BranchID, pgxmock.AnyArg(), &token, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), h)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHubRepo_Update_ErrHubNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewHubRepo(mock)

	h := &hub.Hub{
		ID:             uuid.New().String(),
		BranchID:       nil,
		Metadata:       hub.Metadata{},
		BootstrapToken: "",
		UpdatedAt:      time.Now(),
	}

	mock.ExpectExec("UPDATE hub_registry").
		WithArgs(h.ID, (*uuid.UUID)(nil), pgxmock.AnyArg(), (*string)(nil), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = repo.Update(context.Background(), h)
	assert.ErrorIs(t, err, hub.ErrHubNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHubRepo_Update_ExecError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewHubRepo(mock)

	h := &hub.Hub{
		ID:             uuid.New().String(),
		BranchID:       nil,
		Metadata:       hub.Metadata{},
		BootstrapToken: "",
		UpdatedAt:      time.Now(),
	}

	mock.ExpectExec("UPDATE hub_registry").
		WithArgs(h.ID, (*uuid.UUID)(nil), pgxmock.AnyArg(), (*string)(nil), pgxmock.AnyArg()).
		WillReturnError(errors.New("connection error"))

	err = repo.Update(context.Background(), h)
	assert.ErrorContains(t, err, "update hub")
	assert.ErrorContains(t, err, "connection error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHubRepo_Update_AllFields(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewHubRepo(mock)

	bid := uuid.New()
	token := "secure-token"
	h := &hub.Hub{
		ID:             uuid.New().String(),
		BranchID:       &bid,
		Metadata:       hub.Metadata{HardwareModel: "HW-3000", POSProvider: "POS-9000", FirmwareVersion: "2.1.0"},
		BootstrapToken: token,
		UpdatedAt:      time.Now(),
	}

	mock.ExpectExec("UPDATE hub_registry").
		WithArgs(h.ID, h.BranchID, pgxmock.AnyArg(), &token, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), h)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
