package hub

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/tablehub/cloud/pkg/types"
)

// ConnectionStatus represents the current connectivity state of a hub.
type ConnectionStatus string

const (
	StatusOnline         ConnectionStatus = "online"
	StatusOffline        ConnectionStatus = "offline"
	StatusAuthenticating ConnectionStatus = "authenticating"
)

// Metadata holds hub-specific telemetry stored as JSONB in PostgreSQL.
type Metadata struct {
	Name            string `json:"name,omitempty"`
	FirmwareVersion string `json:"firmware_version,omitempty"`
	HardwareModel   string `json:"hardware_model,omitempty"`
	POSProvider     string `json:"pos_provider,omitempty"`
	IPAddress       string `json:"ip_address,omitempty"`
	OSVersion       string `json:"os_version,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for Metadata.
func (m Metadata) MarshalJSON() ([]byte, error) {
	type Alias Metadata
	return json.Marshal(Alias(m))
}

// Hub represents a physical hub device registered to a organization.
type Hub struct {
	ID               types.HubID          `json:"id"`
	OrganizationID   types.OrganizationID `json:"organization_id"`
	BranchID         *uuid.UUID           `json:"branch_id,omitempty"`
	PublicKey        string               `json:"public_key"`
	ConnectionStatus ConnectionStatus     `json:"connection_status"`
	LastSeen         *time.Time           `json:"last_seen,omitempty"`
	Metadata         Metadata             `json:"metadata"`
	BootstrapToken   string               `json:"-"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// IsOnline returns true if the hub is currently connected.
func (h *Hub) IsOnline() bool {
	return h.ConnectionStatus == StatusOnline
}

// Repository defines the persistence contract for hubs.
// All methods accept a context.Context for cancellation and timeout support.
type Repository interface {
	Register(ctx context.Context, h *Hub) error
	GetByID(ctx context.Context, id types.HubID) (*Hub, error)
	GetByOrganization(ctx context.Context, organizationID types.OrganizationID) ([]*Hub, error)
	GetOnline(ctx context.Context) ([]*Hub, error)
	UpdateStatus(ctx context.Context, id types.HubID, status ConnectionStatus, lastSeen *time.Time) error
	UpdateMetadata(ctx context.Context, id types.HubID, meta Metadata) error
	UpdatePublicKeyAndBurnToken(ctx context.Context, id types.HubID, publicKey string) error
	Update(ctx context.Context, h *Hub) error
	Delete(ctx context.Context, id types.HubID) error
	CountByOrganization(ctx context.Context, organizationID types.OrganizationID) (int, error)
}
