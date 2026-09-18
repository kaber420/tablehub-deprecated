package pendingop

import "errors"

var (
	// ErrNoPendingOps is returned when no operations are available for processing.
	ErrNoPendingOps = errors.New("no pending operations found")

	// ErrOpNotFound is returned when an operation is not found by its ID.
	ErrOpNotFound = errors.New("pending operation not found")
)
