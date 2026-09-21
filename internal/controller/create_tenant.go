package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateTenant registers a Tenant. Admin-only.
type CreateTenant struct {
	Tenants ports.TenantStore
	Clock   ports.Clock
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
	return t, nil
}
