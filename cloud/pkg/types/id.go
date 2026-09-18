package types

import "github.com/google/uuid"

// OrganizationID uniquely identifies a organization tenant.
type OrganizationID = uuid.UUID

// HubID uniquely identifies a hub within a organization.
type HubID = string

// UserID uniquely identifies a user.
type UserID = uuid.UUID

// NewOrganizationID generates a new random UUID for an organization.
func NewOrganizationID() OrganizationID {
	return uuid.New()
}

// NewUserID generates a new random UUID for a user.
func NewUserID() UserID {
	return uuid.New()
}

// ParseOrganizationID parses a UUID string into an OrganizationID.
func ParseOrganizationID(s string) (OrganizationID, error) {
	return uuid.Parse(s)
}
