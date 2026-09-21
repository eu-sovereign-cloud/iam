package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// GetTenant fetches a single Tenant by ID.
type GetTenant struct {
	Tenants ports.TenantStore
}

func (c *GetTenant) Do(ctx context.Context, tenantID string) (model.Tenant, error) {
	return c.Tenants.GetTenant(ctx, tenantID)
}
