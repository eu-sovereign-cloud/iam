package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// SetGrantAdmin toggles a Grant's tenant-admin flag, swapping its ecp
// RoleAssignment between model.TenantAdminRole and the Grant's own Roles
// (see doc/adr/0018). Global-admin-only: a tenant admin cannot promote
// or demote tenant-admin status, even for their own tenant — only a
// global admin can (see doc/adr/0016).
type SetGrantAdmin struct {
	Grants      ports.GrantStore
	TenantRoles ports.TenantRoleStore
}

func (c *SetGrantAdmin) Do(ctx context.Context, subject, tenantID string, admin bool) (model.Grant, error) {
	if err := model.RequireAdmin(ctx); err != nil {
		return model.Grant{}, err
	}
	g, err := c.Grants.GetGrant(ctx, subject, tenantID)
	if err != nil {
		return model.Grant{}, err
	}
	g.Admin = admin
	if err := c.Grants.UpdateGrant(ctx, g); err != nil {
		return model.Grant{}, err
	}
	roles := g.Roles
	if admin {
		roles = []string{model.TenantAdminRole}
	}
	if err := c.TenantRoles.SetRoleAssignment(ctx, tenantID, subject, roles); err != nil {
		return model.Grant{}, err
	}
	return g, nil
}
