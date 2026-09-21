package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// GetTenant fetches a single Tenant by ID.
type GetTenant struct {
	tenants ports.TenantStore
}

func NewGetTenant(tenants ports.TenantStore) *GetTenant {
	return &GetTenant{tenants: tenants}
}

func (c *GetTenant) Do(ctx context.Context, tenantID string) (model.Tenant, error) {
	return c.tenants.GetTenant(ctx, tenantID)
}
