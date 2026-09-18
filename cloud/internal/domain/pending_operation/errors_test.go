package pendingop

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors(t *testing.T) {
	assert.Error(t, ErrNoPendingOps)
	assert.Equal(t, "no pending operations found", ErrNoPendingOps.Error())

	assert.Error(t, ErrOpNotFound)
	assert.Equal(t, "pending operation not found", ErrOpNotFound.Error())
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	assert.NotEqual(t, ErrNoPendingOps, ErrOpNotFound)
}

func TestSentinelErrors_CanBeWrapped(t *testing.T) {
	wrapped := errors.Join(ErrOpNotFound, errors.New("db error"))
	assert.True(t, errors.Is(wrapped, ErrOpNotFound))
	assert.False(t, errors.Is(wrapped, ErrNoPendingOps))
}
