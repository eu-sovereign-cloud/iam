package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteGrant revokes a subject's claim to a Tenant. Callers are
// responsible for enforcing that only admins invoke it.
type DeleteGrant struct {
	grants ports.GrantStore
}

func NewDeleteGrant(grants ports.GrantStore) *DeleteGrant {
	return &DeleteGrant{grants: grants}
}

func (c *DeleteGrant) Do(ctx context.Context, subject, tenantID string) error {
	return c.grants.DeleteGrant(ctx, subject, tenantID)
}
