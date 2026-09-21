package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteTenant removes a Tenant. Callers are responsible for enforcing
// that only admins invoke it.
type DeleteTenant struct {
	tenants ports.TenantStore
}

func NewDeleteTenant(tenants ports.TenantStore) *DeleteTenant {
	return &DeleteTenant{tenants: tenants}
}

func (c *DeleteTenant) Do(ctx context.Context, tenantID string) error {
	return c.tenants.DeleteTenant(ctx, tenantID)
}
