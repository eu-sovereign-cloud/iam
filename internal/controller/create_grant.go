package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateGrant records that a User may claim a Tenant, and binds it to
// roles via a RoleAssignment in ecp (see doc/adr/0018). Global-admin-only,
// or a tenant admin of tenantID (see doc/adr/0016).
type CreateGrant struct {
	Grants      ports.GrantStore
	Users       ports.UserStore
	Tenants     ports.TenantStore
	Clock       ports.Clock
	TenantRoles ports.TenantRoleStore
}

func (c *CreateGrant) Do(ctx context.Context, subject, tenantID string, roles []string, grantedBy string) (model.Grant, error) {
	subject = strings.TrimSpace(subject)
	tenantID = strings.TrimSpace(tenantID)
	roles = trimRoles(roles)
	if err := requireTenantAdmin(ctx, c.Grants, tenantID); err != nil {
		return model.Grant{}, err
	}
	if len(roles) == 0 {
		return model.Grant{}, fmt.Errorf("%w: at least one role is required", model.ErrInvalid)
	}
	if _, err := c.Users.GetUser(ctx, subject); err != nil {
		return model.Grant{}, err
	}
	if _, err := c.Tenants.GetTenant(ctx, tenantID); err != nil {
		return model.Grant{}, err
	}
	g := model.Grant{
		Subject:   subject,
		TenantID:  tenantID,
		GrantedAt: c.Clock.Now(),
		GrantedBy: grantedBy,
		Roles:     roles,
	}
	if err := c.Grants.CreateGrant(ctx, g); err != nil {
		return model.Grant{}, err
	}
	if err := c.TenantRoles.SetRoleAssignment(ctx, tenantID, subject, roles); err != nil {
		return model.Grant{}, err
	}
	return g, nil
}

// trimRoles trims whitespace from each role and drops any that end up
// blank.
func trimRoles(roles []string) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}
