package tunnel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tablehub/cloud/internal/domain/hub"
)

// ProvisionHubUseCase generates a secure, single-use bootstrap token
// for an existing hub and prepares the provisioning file data.
type ProvisionHubUseCase struct {
	hubRepo    hub.Repository
	cloudWSURL string
}

// NewProvisionHubUseCase creates a new ProvisionHubUseCase with its dependencies.
func NewProvisionHubUseCase(hubRepo hub.Repository, cloudWSURL string) *ProvisionHubUseCase {
	return &ProvisionHubUseCase{
		hubRepo:    hubRepo,
		cloudWSURL: cloudWSURL,
	}
}

// ProvisionResult holds the data needed to generate the .thub provisioning file.
type ProvisionResult struct {
	HubID          string `json:"hub_id"`
	OrganizationID string `json:"organization_id"`
	CloudWSURL     string `json:"cloud_url"`
	BootstrapToken string `json:"bootstrap_token"`
	POSProvider    string `json:"pos_provider,omitempty"`
}

// Execute generates a new bootstrap token for an existing hub
// and returns the data needed to generate the .thub provisioning file.
func (uc *ProvisionHubUseCase) Execute(ctx context.Context, organizationID string, hubID string) (*ProvisionResult, error) {
	// 1. Validate organization ID
	_, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization_id: %w", err)
	}

	if hubID == "" {
		return nil, fmt.Errorf("invalid hub_id")
	}

	// 2. Generate secure bootstrap token
	bToken := make([]byte, 32)
	if _, err := rand.Read(bToken); err != nil {
		return nil, fmt.Errorf("generate bootstrap token: %w", err)
	}
	bootstrapToken := hex.EncodeToString(bToken)

	// 3. Fetch existing hub
	h, err := uc.hubRepo.GetByID(ctx, hubID)
	if err != nil {
		return nil, fmt.Errorf("fetch hub: %w", err)
	}

	if h.OrganizationID.String() != organizationID {
		return nil, fmt.Errorf("forbidden: hub does not belong to organization")
	}

	// 4. Update the hub with the new bootstrap token
	h.BootstrapToken = bootstrapToken
	h.UpdatedAt = time.Now()

	if err := uc.hubRepo.Update(ctx, h); err != nil {
		return nil, fmt.Errorf("update hub token: %w", err)
	}

	// 5. Return the provisioning data
	return &ProvisionResult{
		HubID:          hubID,
		OrganizationID: organizationID,
		CloudWSURL:     uc.cloudWSURL,
		BootstrapToken: bootstrapToken,
		POSProvider:    h.Metadata.POSProvider,
	}, nil
}
