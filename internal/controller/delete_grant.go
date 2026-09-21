package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteGrant revokes a subject's claim to a Tenant, and removes its
// RoleAssignment in ecp (see doc/adr/0018). Global-admin-only, or a
// tenant admin of tenantID (see doc/adr/0016).
type DeleteGrant struct {
	Grants      ports.GrantStore
	TenantRoles ports.TenantRoleStore
}

func (c *DeleteGrant) Do(ctx context.Context, subject, tenantID string) error {
	if err := requireTenantAdmin(ctx, c.Grants, tenantID); err != nil {
		return err
	}
	if err := c.Grants.DeleteGrant(ctx, subject, tenantID); err != nil {
		return err
	}
	return c.TenantRoles.DeleteRoleAssignment(ctx, tenantID, subject)
}
