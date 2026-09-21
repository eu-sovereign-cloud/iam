package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteTenant removes a Tenant. Callers are responsible for enforcing
// that only admins invoke it.
type DeleteTenant struct {
	Tenants ports.TenantStore
}

func (c *DeleteTenant) Do(ctx context.Context, tenantID string) error {
	return c.Tenants.DeleteTenant(ctx, tenantID)
}
