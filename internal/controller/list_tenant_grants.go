package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListTenantGrants lists every Grant for a tenant - "who has access to
// this tenant". Tenant-admin-or-global-admin (ADR 0016).
type ListTenantGrants struct {
	Grants ports.GrantStore
}

func (c *ListTenantGrants) Do(ctx context.Context, tenantID string) ([]model.Grant, error) {
	if err := requireTenantAdmin(ctx, c.Grants, tenantID); err != nil {
		return nil, err
	}
	return c.Grants.ListGrantsByTenant(ctx, tenantID)
}
