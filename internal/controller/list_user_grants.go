package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListUserGrants lists the Tenants a subject may claim.
type ListUserGrants struct {
	Grants ports.GrantStore
}

func (c *ListUserGrants) Do(ctx context.Context, subject string) ([]model.Grant, error) {
	return c.Grants.ListGrantsBySubject(ctx, subject)
}
