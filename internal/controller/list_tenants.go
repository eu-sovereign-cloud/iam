package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListTenants lists all registered Tenants. Admin-only.
type ListTenants struct {
	Tenants ports.TenantStore
}

func (c *ListTenants) Do(ctx context.Context) ([]model.Tenant, error) {
	if err := model.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	return c.Tenants.ListTenants(ctx)
}
