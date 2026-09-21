package kubestore

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/pkg/kube"
)

func grantToConfigMap(g model.Grant, namespace string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	cm.Name = grantName(g.Subject, g.TenantID)
	cm.Namespace = namespace
	cm.Labels = map[string]string{
		labelType:   typeGrant,
		labelUser:   shortHash(g.Subject),
		labelTenant: shortHash(g.TenantID),
		labelAdmin:  strconv.FormatBool(g.Admin),
	}
	cm.Data = map[string]string{
		dataSubject:   g.Subject,
		dataTenantID:  g.TenantID,
		dataGrantedAt: g.GrantedAt.Format(time.RFC3339),
		dataGrantedBy: g.GrantedBy,
		dataAdmin:     strconv.FormatBool(g.Admin),
		dataRoles:     strings.Join(g.Roles, ","),
	}
	return cm
}

func grantFromConfigMap(cm *corev1.ConfigMap) model.Grant {
	granted, _ := time.Parse(time.RFC3339, cm.Data[dataGrantedAt])
	admin, _ := strconv.ParseBool(cm.Data[dataAdmin])
	var roles []string
	if raw := cm.Data[dataRoles]; raw != "" {
		roles = strings.Split(raw, ",")
	}
	return model.Grant{
		Subject:   cm.Data[dataSubject],
		TenantID:  cm.Data[dataTenantID],
		GrantedAt: granted,
		GrantedBy: cm.Data[dataGrantedBy],
		Admin:     admin,
		Roles:     roles,
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
	if _, err := s.client.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, kube.MetaCreateOpts()); err != nil {
		if kube.IsAlreadyExists(err) {
			return fmt.Errorf("%w: grant %s/%s", model.ErrConflict, g.Subject, g.TenantID)
		}
		return fmt.Errorf("creating grant %s/%s: %w", g.Subject, g.TenantID, err)
	}

	s.mu.Lock()
	s.grants[key] = g
	s.mu.Unlock()
	return nil
}

func (s *Store) GetGrant(_ context.Context, subject, tenantID string) (model.Grant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.grants[grantKey(subject, tenantID)]
	if !ok {
		return model.Grant{}, fmt.Errorf("%w: grant %s/%s", model.ErrNotFound, subject, tenantID)
	}
	return g, nil
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

func (s *Store) ListGrantsByTenant(_ context.Context, tenantID string) ([]model.Grant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Grant
	for _, g := range s.grants {
		if g.TenantID == tenantID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (s *Store) UpdateGrant(ctx context.Context, g model.Grant) error {
	key := grantKey(g.Subject, g.TenantID)
	s.mu.RLock()
	_, exists := s.grants[key]
	s.mu.RUnlock()
	if !exists {
		return fmt.Errorf("%w: grant %s/%s", model.ErrNotFound, g.Subject, g.TenantID)
	}

	cm := grantToConfigMap(g, s.namespace)
	if _, err := s.client.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, kube.MetaUpdateOpts()); err != nil {
		return fmt.Errorf("updating grant %s/%s: %w", g.Subject, g.TenantID, err)
	}

	s.mu.Lock()
	s.grants[key] = g
	s.mu.Unlock()
	return nil
}

func (s *Store) DeleteGrant(ctx context.Context, subject, tenantID string) error {
	name := grantName(subject, tenantID)
	if err := s.client.CoreV1().ConfigMaps(s.namespace).Delete(ctx, name, kube.MetaDeleteOpts()); err != nil && !kube.IsNotFound(err) {
		return fmt.Errorf("deleting grant %s/%s: %w", subject, tenantID, err)
	}

	s.mu.Lock()
	delete(s.grants, grantKey(subject, tenantID))
	s.mu.Unlock()
	return nil
}
