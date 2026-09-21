package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListUserGrants lists the Tenants a subject may claim.
type ListUserGrants struct {
	grants ports.GrantStore
}

func NewListUserGrants(grants ports.GrantStore) *ListUserGrants {
	return &ListUserGrants{grants: grants}
}

func (c *ListUserGrants) Do(ctx context.Context, subject string) ([]model.Grant, error) {
	return c.grants.ListGrantsBySubject(ctx, subject)
}
