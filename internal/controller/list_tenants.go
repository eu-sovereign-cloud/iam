package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListTenants lists all registered Tenants.
type ListTenants struct {
	tenants ports.TenantStore
}

func NewListTenants(tenants ports.TenantStore) *ListTenants {
	return &ListTenants{tenants: tenants}
}

func (c *ListTenants) Do(ctx context.Context) ([]model.Tenant, error) {
	return c.tenants.ListTenants(ctx)
}
