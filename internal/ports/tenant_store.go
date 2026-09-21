package ports

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// TenantStore is the persistence port for Tenants.
type TenantStore interface {
	CreateTenant(ctx context.Context, t model.Tenant) error
	GetTenant(ctx context.Context, tenantID string) (model.Tenant, error)
	ListTenants(ctx context.Context) ([]model.Tenant, error)
	DeleteTenant(ctx context.Context, tenantID string) error
}
