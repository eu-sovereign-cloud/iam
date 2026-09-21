package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// requireTenantAdmin returns model.ErrForbidden unless the caller in ctx
// is a global admin, or holds an admin Grant for tenantID. Unlike
// model.RequireAdmin/RequireSelfOrAdmin, this isn't a pure function of
// ctx — tenant-admin status lives on a Grant (see doc/adr/0016), not on
// the identity model.WithIdentity already stashed in ctx at auth time —
// so it needs a GrantStore lookup, and belongs here rather than in
// internal/model (which must not depend on internal/ports).
func requireTenantAdmin(ctx context.Context, grants ports.GrantStore, tenantID string) error {
	caller := model.IdentityFromContext(ctx)
	if caller.Admin {
		return nil
	}
	g, err := grants.GetGrant(ctx, caller.Subject, tenantID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return fmt.Errorf("%w: tenant admin privileges required for tenant %s", model.ErrForbidden, tenantID)
		}
		return err
	}
	if !g.Admin {
		return fmt.Errorf("%w: tenant admin privileges required for tenant %s", model.ErrForbidden, tenantID)
	}
	return nil
}
