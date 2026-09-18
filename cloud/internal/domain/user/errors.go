package user

import (
	"errors"
)

var (
	// ErrNotFound is returned when a user does not exist.
	ErrNotFound = errors.New("user not found")

	// ErrDuplicateProviderUserID is returned when attempting to create a user
	// with a provider_user_id that already exists.
	ErrDuplicateProviderUserID = errors.New("provider user id already exists")
)
