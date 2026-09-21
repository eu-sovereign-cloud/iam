package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteGrant revokes a subject's claim to a Tenant. Callers are
// responsible for enforcing that only admins invoke it.
type DeleteGrant struct {
	Grants ports.GrantStore
}

func (c *DeleteGrant) Do(ctx context.Context, subject, tenantID string) error {
	return c.Grants.DeleteGrant(ctx, subject, tenantID)
}
