package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteGrant revokes a subject's claim to a Tenant. Admin-only.
type DeleteGrant struct {
	Grants ports.GrantStore
}

func (c *DeleteGrant) Do(ctx context.Context, subject, tenantID string) error {
	if err := model.RequireAdmin(ctx); err != nil {
		return err
	}
	return c.Grants.DeleteGrant(ctx, subject, tenantID)
}
