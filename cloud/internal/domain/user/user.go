package user

import (
	"time"

	"github.com/tablehub/cloud/pkg/types"
)

// Role defines the authorization level of a user within an organization.
type Role string

const (
	// Staff roles (platform-level, no organization)
	RolePlatformAdmin Role = "platform_admin"
	RoleSupport       Role = "support"
	RoleBilling       Role = "billing"

	// Client roles (organization-scoped)
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

// IsStaffRole returns true if the role belongs to platform staff.
func (r Role) IsStaffRole() bool {
	switch r {
	case RolePlatformAdmin, RoleSupport, RoleBilling:
		return true
	}
	return false
}

// User represents a human user linked to an organization via identity provider.
type User struct {
	ID              types.UserID         `json:"id"`
	ProviderUserID  string               `json:"provider_user_id"`
	OrganizationID  types.OrganizationID `json:"organization_id"`
	Role            Role                 `json:"role"`
	OnboardingState OnboardingState      `json:"onboarding_state"`
	Name            string               `json:"name"`
	Email           string               `json:"email"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// OnboardingState defines the current onboarding phase of a user.
type OnboardingState string

const (
	StateNew          OnboardingState = "new"
	StateProfileSetup OnboardingState = "profile_setup"
	StateOrgPending   OnboardingState = "org_pending"
	StateOrgCreating  OnboardingState = "org_creating"
	StateOrgFailed    OnboardingState = "org_failed"
	StateActive       OnboardingState = "active"
)
