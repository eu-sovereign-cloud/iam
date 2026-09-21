package controller

import (
	"context"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateGrant records that a User may claim a Tenant. Callers are
// responsible for enforcing that only admins invoke it.
//
// TODO(eu-sovereign-cloud/iam#future): when a Grant is created or deleted,
// this is the place to eventually notify ecp so it can create/remove the
// corresponding RoleAssignment in the tenant's namespace. Not implemented:
// IAM has no ecp credentials/access yet and the IAM-grant -> ecp-role
// mapping isn't defined. See ADR 0008.
type CreateGrant struct {
	grants  ports.GrantStore
	users   ports.UserStore
	tenants ports.TenantStore
	clock   ports.Clock
}

func NewCreateGrant(grants ports.GrantStore, users ports.UserStore, tenants ports.TenantStore, clock ports.Clock) *CreateGrant {
	return &CreateGrant{grants: grants, users: users, tenants: tenants, clock: clock}
}

func (c *CreateGrant) Do(ctx context.Context, subject, tenantID, grantedBy string) (model.Grant, error) {
	subject = strings.TrimSpace(subject)
	tenantID = strings.TrimSpace(tenantID)
	if _, err := c.users.GetUser(ctx, subject); err != nil {
		return model.Grant{}, err
	}
	if _, err := c.tenants.GetTenant(ctx, tenantID); err != nil {
		return model.Grant{}, err
	}
	g := model.Grant{
		Subject:   subject,
		TenantID:  tenantID,
		GrantedAt: c.clock.Now(),
		GrantedBy: grantedBy,
	}
	if err := c.grants.CreateGrant(ctx, g); err != nil {
		return model.Grant{}, err
	}
	return g, nil
}
