package organization

import "errors"

var (
	// ErrNotFound is returned when a organization does not exist.
	ErrNotFound = errors.New("organization not found")

	// ErrDuplicateSlug is returned when attempting to create a organization
	// with a slug that is already taken.
	ErrDuplicateSlug = errors.New("organization slug already exists")

	// ErrInvalidPlan is returned when an unrecognized plan value is used.
	ErrInvalidPlan = errors.New("invalid organization plan")
)
