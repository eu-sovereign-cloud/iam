package adapter

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func tenantToConfigMap(t model.Tenant, namespace string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	cm.Name = tenantName(t.TenantID)
	cm.Namespace = namespace
	cm.Labels = map[string]string{labelType: typeTenant}
	cm.Data = map[string]string{
		dataTenantID:    t.TenantID,
		dataDisplayName: t.DisplayName,
		dataCreatedAt:   t.CreatedAt.Format(time.RFC3339),
	}
	return cm
}

func tenantFromConfigMap(cm *corev1.ConfigMap) model.Tenant {
	created, _ := time.Parse(time.RFC3339, cm.Data[dataCreatedAt])
	return model.Tenant{
		TenantID:    cm.Data[dataTenantID],
		DisplayName: cm.Data[dataDisplayName],
		CreatedAt:   created,
	}
}

func (s *Store) CreateTenant(ctx context.Context, t model.Tenant) error {
	s.mu.RLock()
	_, exists := s.tenants[t.TenantID]
	s.mu.RUnlock()
	if exists {
		return fmt.Errorf("%w: tenant %q", model.ErrConflict, t.TenantID)
	}

	cm := tenantToConfigMap(t, s.namespace)
	if _, err := s.client.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metaCreateOpts()); err != nil {
		if isAlreadyExists(err) {
			return fmt.Errorf("%w: tenant %q", model.ErrConflict, t.TenantID)
		}
		return fmt.Errorf("creating tenant %q: %w", t.TenantID, err)
	}

	s.mu.Lock()
	s.tenants[t.TenantID] = t
	s.mu.Unlock()
	return nil
}

func (s *Store) GetTenant(_ context.Context, tenantID string) (model.Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[tenantID]
	if !ok {
		return model.Tenant{}, fmt.Errorf("%w: tenant %q", model.ErrNotFound, tenantID)
	}
	return t, nil
}

func (s *Store) ListTenants(_ context.Context) ([]model.Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) DeleteTenant(ctx context.Context, tenantID string) error {
	name := tenantName(tenantID)
	if err := s.client.CoreV1().ConfigMaps(s.namespace).Delete(ctx, name, metaDeleteOpts()); err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting tenant %q: %w", tenantID, err)
	}

	s.mu.Lock()
	delete(s.tenants, tenantID)
	s.mu.Unlock()
	return nil
}
