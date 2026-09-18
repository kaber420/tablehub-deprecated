package webhook

import "errors"

var (
	ErrInvalidToken        = errors.New("invalid webhook token")
	ErrUnsupportedProvider = errors.New("unsupported webhook provider")
	ErrEventNotFound       = errors.New("webhook event not found")
	ErrDuplicateEvent      = errors.New("duplicate webhook event")
	ErrNoPendingEvents     = errors.New("no pending webhook events")
)
