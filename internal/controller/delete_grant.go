package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteGrant revokes a subject's claim to a Tenant. Global-admin-only,
// or a tenant admin of tenantID (see doc/adr/0016).
type DeleteGrant struct {
	Grants ports.GrantStore
}

func (c *DeleteGrant) Do(ctx context.Context, subject, tenantID string) error {
	if err := requireTenantAdmin(ctx, c.Grants, tenantID); err != nil {
		return err
	}
	return c.Grants.DeleteGrant(ctx, subject, tenantID)
}
