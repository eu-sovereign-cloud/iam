package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteTenant removes a Tenant. Admin-only.
type DeleteTenant struct {
	Tenants ports.TenantStore
}

func (c *DeleteTenant) Do(ctx context.Context, tenantID string) error {
	if err := model.RequireAdmin(ctx); err != nil {
		return err
	}
	return c.Tenants.DeleteTenant(ctx, tenantID)
}
