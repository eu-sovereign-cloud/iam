package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateTenant registers a Tenant. Admin-only.
//
// ecp has no Tenant entity of its own — its Kubernetes namespace is
// auto-provisioned the first time a tenant-scoped object is written into
// it (see doc/adr/0018), so creating the tenant-admin Role is what
// actually "creates the tenant" on the ecp side.
type CreateTenant struct {
	Tenants     ports.TenantStore
	Clock       ports.Clock
	TenantRoles ports.TenantRoleStore
}

func (c *CreateTenant) Do(ctx context.Context, tenantID, displayName string) (model.Tenant, error) {
	if err := model.RequireAdmin(ctx); err != nil {
		return model.Tenant{}, err
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return model.Tenant{}, fmt.Errorf("%w: tenantId is required", model.ErrInvalid)
	}
	t := model.Tenant{
		TenantID:    tenantID,
		DisplayName: strings.TrimSpace(displayName),
		CreatedAt:   c.Clock.Now(),
	}
	if err := c.Tenants.CreateTenant(ctx, t); err != nil {
		return model.Tenant{}, err
	}
	if err := c.TenantRoles.EnsureTenantAdminRole(ctx, tenantID); err != nil {
		return model.Tenant{}, err
	}
	return t, nil
}
