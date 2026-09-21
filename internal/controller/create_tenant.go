package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateTenant registers a Tenant. Callers (internal/service, internal/web)
// are responsible for enforcing that only admins invoke it.
type CreateTenant struct {
	Tenants ports.TenantStore
	Clock   ports.Clock
}

func (c *CreateTenant) Do(ctx context.Context, tenantID, displayName string) (model.Tenant, error) {
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
