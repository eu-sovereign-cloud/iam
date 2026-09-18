package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func patToSecret(p model.PAT, namespace string) (*corev1.Secret, error) {
	sec := &corev1.Secret{}
	sec.Name = patName(p.ID)
	sec.Namespace = namespace
	sec.Type = corev1.SecretTypeOpaque
	sec.Labels = map[string]string{
		labelType: typePAT,
		labelUser: shortHash(p.Subject),
	}
	sec.Annotations = map[string]string{
		"id":          p.ID,
		dataSubject:   p.Subject,
		dataName:      p.Name,
		dataCreatedAt: p.CreatedAt.Format(time.RFC3339),
	}
	if p.ExpiresAt != nil {
		sec.Annotations[dataExpiresAt] = p.ExpiresAt.Format(time.RFC3339)
	}
	if p.Scope != nil {
		scopeJSON, err := json.Marshal(p.Scope)
		if err != nil {
			return nil, fmt.Errorf("marshaling PAT scope: %w", err)
		}
		sec.Annotations[dataScope] = string(scopeJSON)
	}
	sec.Data = map[string][]byte{
		dataTokenHash: []byte(p.TokenHash),
	}
	return sec, nil
}

func patFromSecret(sec *corev1.Secret) (model.PAT, error) {
	created, _ := time.Parse(time.RFC3339, sec.Annotations[dataCreatedAt])
	p := model.PAT{
		ID:        sec.Annotations["id"],
		Subject:   sec.Annotations[dataSubject],
		Name:      sec.Annotations[dataName],
		TokenHash: string(sec.Data[dataTokenHash]),
		CreatedAt: created,
	}
	if raw, ok := sec.Annotations[dataExpiresAt]; ok {
		exp, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return model.PAT{}, fmt.Errorf("parsing expires-at: %w", err)
		}
		p.ExpiresAt = &exp
	}
	if raw, ok := sec.Annotations[dataScope]; ok {
		var scope model.TokenScope
		if err := json.Unmarshal([]byte(raw), &scope); err != nil {
			return model.PAT{}, fmt.Errorf("parsing scope: %w", err)
		}
		p.Scope = &scope
	}
	return p, nil
}

func (s *Store) CreatePAT(ctx context.Context, p model.PAT) error {
	sec, err := patToSecret(p, s.namespace)
	if err != nil {
		return err
	}
	if _, err := s.client.CoreV1().Secrets(s.namespace).Create(ctx, sec, metaCreateOpts()); err != nil {
		if isAlreadyExists(err) {
			return fmt.Errorf("%w: PAT %q", model.ErrConflict, p.ID)
		}
		return fmt.Errorf("creating PAT %q: %w", p.ID, err)
	}

	s.mu.Lock()
	s.pats[p.ID] = p
	s.patsByHash[p.TokenHash] = p.ID
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

func (s *Store) GetPATByHash(_ context.Context, tokenHash string) (model.PAT, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.patsByHash[tokenHash]
	if !ok {
		return model.PAT{}, fmt.Errorf("%w: PAT", model.ErrNotFound)
	}
	return s.pats[id], nil
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
	if err := s.client.CoreV1().Secrets(s.namespace).Delete(ctx, name, metaDeleteOpts()); err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting PAT %q: %w", id, err)
	}

	s.mu.Lock()
	if p, ok := s.pats[id]; ok {
		delete(s.patsByHash, p.TokenHash)
	}
	delete(s.pats, id)
	s.mu.Unlock()
	return nil
}
