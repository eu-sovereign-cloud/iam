package adapter

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func grantToConfigMap(g model.Grant, namespace string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	cm.Name = grantName(g.Subject, g.TenantID)
	cm.Namespace = namespace
	cm.Labels = map[string]string{
		labelType:   typeGrant,
		labelUser:   shortHash(g.Subject),
		labelTenant: shortHash(g.TenantID),
	}
	cm.Data = map[string]string{
		dataSubject:   g.Subject,
		dataTenantID:  g.TenantID,
		dataGrantedAt: g.GrantedAt.Format(time.RFC3339),
		dataGrantedBy: g.GrantedBy,
	}
	return cm
}

func grantFromConfigMap(cm *corev1.ConfigMap) model.Grant {
	granted, _ := time.Parse(time.RFC3339, cm.Data[dataGrantedAt])
	return model.Grant{
		Subject:   cm.Data[dataSubject],
		TenantID:  cm.Data[dataTenantID],
		GrantedAt: granted,
		GrantedBy: cm.Data[dataGrantedBy],
	}
}

func (s *Store) CreateGrant(ctx context.Context, g model.Grant) error {
	key := grantKey(g.Subject, g.TenantID)
	s.mu.RLock()
	_, exists := s.grants[key]
	s.mu.RUnlock()
	if exists {
		return fmt.Errorf("%w: grant %s/%s", model.ErrConflict, g.Subject, g.TenantID)
	}

	cm := grantToConfigMap(g, s.namespace)
	if _, err := s.client.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metaCreateOpts()); err != nil {
		if isAlreadyExists(err) {
			return fmt.Errorf("%w: grant %s/%s", model.ErrConflict, g.Subject, g.TenantID)
		}
		return fmt.Errorf("creating grant %s/%s: %w", g.Subject, g.TenantID, err)
	}

	s.mu.Lock()
	s.grants[key] = g
	s.mu.Unlock()
	return nil
}

func (s *Store) ListGrantsBySubject(_ context.Context, subject string) ([]model.Grant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Grant
	for _, g := range s.grants {
		if g.Subject == subject {
			out = append(out, g)
		}
	}
	return out, nil
}

func (s *Store) DeleteGrant(ctx context.Context, subject, tenantID string) error {
	name := grantName(subject, tenantID)
	if err := s.client.CoreV1().ConfigMaps(s.namespace).Delete(ctx, name, metaDeleteOpts()); err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting grant %s/%s: %w", subject, tenantID, err)
	}

	s.mu.Lock()
	delete(s.grants, grantKey(subject, tenantID))
	s.mu.Unlock()
	return nil
}
