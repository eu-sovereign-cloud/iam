package service

import (
	"context"
	"fmt"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// TenantService manages Tenants. Callers (the controller layer) are
// responsible for enforcing that only admins invoke the mutating methods.
type TenantService struct {
	store TenantStore
	clock Clock
}

func NewTenantService(store TenantStore, clock Clock) *TenantService {
	return &TenantService{store: store, clock: clock}
}

func (s *TenantService) Create(ctx context.Context, tenantID, displayName string) (model.Tenant, error) {
	if tenantID == "" {
		return model.Tenant{}, fmt.Errorf("%w: tenantId is required", model.ErrInvalid)
	}
	t := model.Tenant{
		TenantID:    tenantID,
		DisplayName: displayName,
		CreatedAt:   s.clock.Now(),
	}
	if err := s.store.CreateTenant(ctx, t); err != nil {
		return model.Tenant{}, err
	}
	return t, nil
}

func (s *TenantService) Get(ctx context.Context, tenantID string) (model.Tenant, error) {
	return s.store.GetTenant(ctx, tenantID)
}

func (s *TenantService) List(ctx context.Context) ([]model.Tenant, error) {
	return s.store.ListTenants(ctx)
}

func (s *TenantService) Delete(ctx context.Context, tenantID string) error {
	return s.store.DeleteTenant(ctx, tenantID)
}
