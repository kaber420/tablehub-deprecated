package db

import (
	"context"
	"encoding/hex"
)

// DBKeyProvider implements ws.KeyProvider by fetching hub public keys from the database.
type DBKeyProvider struct {
	hubRepo *HubRepo
}

// NewDBKeyProvider creates a new DBKeyProvider backed by the given hub repository.
func NewDBKeyProvider(hubRepo *HubRepo) *DBKeyProvider {
	return &DBKeyProvider{hubRepo: hubRepo}
}

// GetHubInfo retrieves the Ed25519 public key and organization ID for the given hub from the database.
func (p *DBKeyProvider) GetHubInfo(hubID string) ([]byte, string, error) {
	h, err := p.hubRepo.GetByID(context.Background(), hubID)
	if err != nil {
		return nil, "", err
	}
	pubKeyBytes, err := hex.DecodeString(h.PublicKey)
	if err != nil {
		return nil, "", err
	}
	return pubKeyBytes, h.OrganizationID.String(), nil
}
