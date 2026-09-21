package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateTenant registers a Tenant. Callers (internal/service, internal/web)
// are responsible for enforcing that only admins invoke it.
type CreateTenant struct {
	tenants ports.TenantStore
	clock   ports.Clock
}

func NewCreateTenant(tenants ports.TenantStore, clock ports.Clock) *CreateTenant {
	return &CreateTenant{tenants: tenants, clock: clock}
}

func (c *CreateTenant) Do(ctx context.Context, tenantID, displayName string) (model.Tenant, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return model.Tenant{}, fmt.Errorf("%w: tenantId is required", model.ErrInvalid)
	}
	t := model.Tenant{
		TenantID:    tenantID,
		DisplayName: strings.TrimSpace(displayName),
		CreatedAt:   c.clock.Now(),
	}
	if err := c.tenants.CreateTenant(ctx, t); err != nil {
		return model.Tenant{}, err
	}
	return t, nil
}
