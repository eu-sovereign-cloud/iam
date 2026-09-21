package controller

import (
	"context"
	"fmt"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteTenant removes a Tenant. Admin-only. Refuses (ErrConflict) while
// any Grant still exists for the tenant — IAM has no visibility into
// whatever else ecp's tenant namespace might still hold (Workspaces,
// workloads, ...), so the only "is this tenant empty" check IAM can make
// for itself is that it holds no more Grants for it (see doc/adr/0018).
// It only ever deletes the tenant-admin Role it manages, never ecp's
// namespace itself — ecp's own auto-cleanup handles that once truly
// empty.
type DeleteTenant struct {
	Tenants     ports.TenantStore
	Grants      ports.GrantStore
	TenantRoles ports.TenantRoleStore
}

func (c *DeleteTenant) Do(ctx context.Context, tenantID string) error {
	if err := model.RequireAdmin(ctx); err != nil {
		return err
	}
	grants, err := c.Grants.ListGrantsByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(grants) > 0 {
		return fmt.Errorf("%w: tenant %q still has %d grant(s)", model.ErrConflict, tenantID, len(grants))
	}
	if err := c.TenantRoles.DeleteTenantAdminRole(ctx, tenantID); err != nil {
		return err
	}
	return c.Tenants.DeleteTenant(ctx, tenantID)
}
