package controller

import (
	"context"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateGrant records that a User may claim a Tenant. Admin-only.
//
// TODO(eu-sovereign-cloud/iam#future): when a Grant is created or deleted,
// this is the place to eventually notify ecp so it can create/remove the
// corresponding RoleAssignment in the tenant's namespace. Not implemented:
// IAM has no ecp credentials/access yet and the IAM-grant -> ecp-role
// mapping isn't defined. See ADR 0008.
type CreateGrant struct {
	Grants  ports.GrantStore
	Users   ports.UserStore
	Tenants ports.TenantStore
	Clock   ports.Clock
}

func (c *CreateGrant) Do(ctx context.Context, subject, tenantID, grantedBy string) (model.Grant, error) {
	if err := model.RequireAdmin(ctx); err != nil {
		return model.Grant{}, err
	}
	subject = strings.TrimSpace(subject)
	tenantID = strings.TrimSpace(tenantID)
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
	}
	if err := c.Grants.CreateGrant(ctx, g); err != nil {
		return model.Grant{}, err
	}
	return g, nil
}
