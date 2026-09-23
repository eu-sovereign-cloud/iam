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
// or a tenant admin of tenantID (see doc/adr/0016) — except creating the
// grant *as* tenant admin (admin=true), which is always global-admin-only,
// same as SetGrantAdmin.
type CreateGrant struct {
	Grants      ports.GrantStore
	Users       ports.UserStore
	Tenants     ports.TenantStore
	Clock       ports.Clock
	TenantRoles ports.TenantRoleStore
}

func (c *CreateGrant) Do(ctx context.Context, subject, tenantID string, roles []string, grantedBy string, admin bool) (model.Grant, error) {
	subject = strings.TrimSpace(subject)
	tenantID = strings.TrimSpace(tenantID)
	roles = trimRoles(roles)

	if admin {
		if err := model.RequireAdmin(ctx); err != nil {
			return model.Grant{}, err
		}
	} else if err := requireTenantAdmin(ctx, c.Grants, tenantID); err != nil {
		return model.Grant{}, err
	}
	// Roles only matter for a non-admin grant: SetGrantAdmin already
	// overwrites the ecp RoleAssignment to model.TenantAdminRole
	// regardless of stored roles, so requiring one here just to satisfy
	// validation before immediately discarding it would be pointless.
	if !admin && len(roles) == 0 {
		return model.Grant{}, fmt.Errorf("%w: at least one role is required", model.ErrInvalid)
	}
	for _, r := range roles {
		if err := model.ValidateDNS1123Label("role", r); err != nil {
			return model.Grant{}, err
		}
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
		Admin:     admin,
	}
	if err := c.Grants.CreateGrant(ctx, g); err != nil {
		return model.Grant{}, err
	}
	assignRoles := roles
	if admin {
		assignRoles = []string{model.TenantAdminRole}
	}
	if err := c.TenantRoles.SetRoleAssignment(ctx, tenantID, subject, assignRoles); err != nil {
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
