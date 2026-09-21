package controller

import (
	"context"
	"errors"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// RepairTenant re-applies IAM's canonical ecp RBAC state for one tenant —
// the tenant-admin Role plus a RoleAssignment for every current Grant —
// overwriting whatever drift may exist, and deletes any IAM-managed
// RoleAssignment that no longer corresponds to a Grant (see
// doc/adr/0018). Global-admin-only, or a tenant admin of tenantID (see
// doc/adr/0016) — the same scope as CreateGrant/DeleteGrant.
type RepairTenant struct {
	Tenants     ports.TenantStore
	Grants      ports.GrantStore
	TenantRoles ports.TenantRoleStore
}

func (c *RepairTenant) Do(ctx context.Context, tenantID string) error {
	if err := requireTenantAdmin(ctx, c.Grants, tenantID); err != nil {
		return err
	}
	if _, err := c.Tenants.GetTenant(ctx, tenantID); err != nil {
		return err
	}

	var errs []error
	if err := c.TenantRoles.EnsureTenantAdminRole(ctx, tenantID); err != nil {
		errs = append(errs, err)
	}

	grants, err := c.Grants.ListGrantsByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	want := make(map[string]bool, len(grants))
	for _, g := range grants {
		want[g.Subject] = true
		roles := g.Roles
		if g.Admin {
			roles = []string{model.TenantAdminRole}
		}
		if err := c.TenantRoles.SetRoleAssignment(ctx, tenantID, g.Subject, roles); err != nil {
			errs = append(errs, err)
		}
	}

	existing, err := c.TenantRoles.ListRoleAssignments(ctx, tenantID)
	if err != nil {
		errs = append(errs, err)
	}
	for _, ra := range existing {
		if !want[ra.Subject] {
			if err := c.TenantRoles.DeleteRoleAssignment(ctx, tenantID, ra.Subject); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}
