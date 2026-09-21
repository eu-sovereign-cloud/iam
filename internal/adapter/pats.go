package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// PATs carry no secret material of their own — IAM never persists the raw
// JWT or a hash of it, only bookkeeping metadata (ADR 0012) — so, unlike
// the earlier opaque-token design, they're stored as ConfigMaps rather
// than Secrets (ADR 0001).
func patToConfigMap(p model.PAT, namespace string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	cm.Name = patName(p.ID)
	cm.Namespace = namespace
	cm.Labels = map[string]string{
		labelType: typePAT,
		labelUser: shortHash(p.Subject),
	}
	cm.Data = map[string]string{
		"id":          p.ID,
		dataSubject:   p.Subject,
		dataName:      p.Name,
		dataCreatedAt: p.CreatedAt.Format(time.RFC3339),
		dataExpiresAt: p.ExpiresAt.Format(time.RFC3339),
	}
	if p.Scope != nil {
		scopeJSON, err := json.Marshal(p.Scope)
		if err != nil {
			return nil, fmt.Errorf("marshaling PAT scope: %w", err)
		}
		cm.Data[dataScope] = string(scopeJSON)
	}
	return cm, nil
}

func patFromConfigMap(cm *corev1.ConfigMap) (model.PAT, error) {
	created, _ := time.Parse(time.RFC3339, cm.Data[dataCreatedAt])
	expires, err := time.Parse(time.RFC3339, cm.Data[dataExpiresAt])
	if err != nil {
		return model.PAT{}, fmt.Errorf("parsing expires-at: %w", err)
	}
	p := model.PAT{
		ID:        cm.Data["id"],
		Subject:   cm.Data[dataSubject],
		Name:      cm.Data[dataName],
		CreatedAt: created,
		ExpiresAt: expires,
	}
	if raw, ok := cm.Data[dataScope]; ok {
		var scope model.TokenScope
		if err := json.Unmarshal([]byte(raw), &scope); err != nil {
			return model.PAT{}, fmt.Errorf("parsing scope: %w", err)
		}
		p.Scope = &scope
	}
	return p, nil
}

func (s *Store) CreatePAT(ctx context.Context, p model.PAT) error {
	s.mu.RLock()
	if _, exists := s.pats[p.ID]; exists {
		s.mu.RUnlock()
		return fmt.Errorf("%w: PAT %q", model.ErrConflict, p.ID)
	}
	if p.Name != "" {
		for _, existing := range s.pats {
			if existing.Subject == p.Subject && existing.Name == p.Name {
				s.mu.RUnlock()
				return fmt.Errorf("%w: PAT named %q for %s", model.ErrConflict, p.Name, p.Subject)
			}
		}
	}
	s.mu.RUnlock()

	cm, err := patToConfigMap(p, s.namespace)
	if err != nil {
		return err
	}
	if _, err := s.client.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, metaCreateOpts()); err != nil {
		if isAlreadyExists(err) {
			return fmt.Errorf("%w: PAT %q", model.ErrConflict, p.ID)
		}
		return fmt.Errorf("creating PAT %q: %w", p.ID, err)
	}

	s.mu.Lock()
	s.pats[p.ID] = p
	s.mu.Unlock()
	return nil
}

func (s *Store) GetPAT(_ context.Context, id string) (model.PAT, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pats[id]
	if !ok {
		return model.PAT{}, fmt.Errorf("%w: PAT %q", model.ErrNotFound, id)
	}
	return p, nil
}

func (s *Store) ListPATsBySubject(_ context.Context, subject string) ([]model.PAT, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.PAT
	for _, p := range s.pats {
		if p.Subject == subject {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Store) DeletePAT(ctx context.Context, id string) error {
	name := patName(id)
	if err := s.client.CoreV1().ConfigMaps(s.namespace).Delete(ctx, name, metaDeleteOpts()); err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting PAT %q: %w", id, err)
	}

	s.mu.Lock()
	delete(s.pats, id)
	s.mu.Unlock()
	return nil
}
