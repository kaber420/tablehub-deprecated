package branch

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/tablehub/cloud/pkg/types"
)

type BranchID uuid.UUID

func (id BranchID) String() string {
	return uuid.UUID(id).String()
}

func (id BranchID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(id).String())
}

func (id *BranchID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*id = BranchID(parsed)
	return nil
}

type Branch struct {
	ID             BranchID             `json:"id"`
	OrganizationID types.OrganizationID `json:"organization_id"`
	Name           string               `json:"name"`
	Address        string               `json:"address"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

type Repository interface {
	Create(ctx context.Context, b *Branch) error
	GetByID(ctx context.Context, id BranchID) (*Branch, error)
	ListByOrganization(ctx context.Context, orgID types.OrganizationID) ([]*Branch, error)
	Update(ctx context.Context, b *Branch) error
	Delete(ctx context.Context, id BranchID) error
	CountByOrganization(ctx context.Context, orgID types.OrganizationID) (int, error)
}
