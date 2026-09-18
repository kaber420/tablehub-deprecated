package organization

import (
	"context"
	"time"

	"github.com/tablehub/cloud/pkg/types"
)

// Plan represents the subscription tier for a organization.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanStarter    Plan = "starter"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

// ValidPlans is the set of recognized plan values.
var ValidPlans = map[Plan]bool{
	PlanFree:       true,
	PlanStarter:    true,
	PlanPro:        true,
	PlanEnterprise: true,
}

// Settings holds organization-specific configuration stored as JSONB.
type Settings struct {
	WebhookToken string `json:"webhook_token,omitempty"`
	Timezone     string `json:"timezone,omitempty"`
	Language     string `json:"language,omitempty"`
}

// Organization represents a organization tenant in the cloud system.
type Organization struct {
	ID            types.OrganizationID `json:"id"`
	ProviderOrgID string               `json:"provider_org_id"`
	Slug          string               `json:"slug"`
	Name          string               `json:"name"`
	Plan          Plan                 `json:"plan"`
	Settings      Settings             `json:"settings"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// Repository defines the persistence contract for organizations.
// Infrastructure implementations (PostgreSQL, etc.) implement this interface.
type Repository interface {
	Create(ctx context.Context, r *Organization) error
	GetByID(ctx context.Context, id types.OrganizationID) (*Organization, error)
	GetBySlug(ctx context.Context, slug string) (*Organization, error)
	List(ctx context.Context, limit, offset int) ([]*Organization, error)
	Update(ctx context.Context, r *Organization) error
	Delete(ctx context.Context, id types.OrganizationID) error
}
