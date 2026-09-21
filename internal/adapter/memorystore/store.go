// Package memorystore is a non-persistent, in-memory implementation of
// every store port (ports.UserStore, TenantStore, GrantStore, PATStore),
// for tests that need a real implementation of those ports without
// standing up a fake Kubernetes clientset. See kubestore for the
// production, Kubernetes-ConfigMap/Secret-backed implementation this
// exists alongside.
package memorystore

import (
	"context"
	"fmt"
	"sync"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type Store struct {
	mu sync.Mutex

	users   map[string]model.User
	tenants map[string]model.Tenant
	grants  map[string]model.Grant
	pats    map[string]model.PAT
}

func New() *Store {
	return &Store{
		users:   map[string]model.User{},
		tenants: map[string]model.Tenant{},
		grants:  map[string]model.Grant{},
		pats:    map[string]model.PAT{},
	}
}

func (s *Store) CreateUser(_ context.Context, u model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[u.Subject]; ok {
		return fmt.Errorf("%w: %s", model.ErrConflict, u.Subject)
	}
	s.users[u.Subject] = u
	return nil
}

func (s *Store) GetUser(_ context.Context, subject string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[subject]
	if !ok {
		return model.User{}, fmt.Errorf("%w: %s", model.ErrNotFound, subject)
	}
	return u, nil
}

func (s *Store) ListUsers(_ context.Context) ([]model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out, nil
}

func (s *Store) UpdateUser(_ context.Context, u model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[u.Subject]; !ok {
		return fmt.Errorf("%w: %s", model.ErrNotFound, u.Subject)
	}
	s.users[u.Subject] = u
	return nil
}

func (s *Store) DeleteUser(_ context.Context, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, subject)
	return nil
}

func (s *Store) CreateTenant(_ context.Context, t model.Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[t.TenantID]; ok {
		return fmt.Errorf("%w: %s", model.ErrConflict, t.TenantID)
	}
	s.tenants[t.TenantID] = t
	return nil
}

func (s *Store) GetTenant(_ context.Context, tenantID string) (model.Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[tenantID]
	if !ok {
		return model.Tenant{}, fmt.Errorf("%w: %s", model.ErrNotFound, tenantID)
	}
	return t, nil
}

func (s *Store) ListTenants(_ context.Context) ([]model.Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) DeleteTenant(_ context.Context, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tenants, tenantID)
	return nil
}

func grantKey(subject, tenantID string) string { return subject + "|" + tenantID }

func (s *Store) CreateGrant(_ context.Context, g model.Grant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := grantKey(g.Subject, g.TenantID)
	if _, ok := s.grants[key]; ok {
		return fmt.Errorf("%w: %s", model.ErrConflict, key)
	}
	s.grants[key] = g
	return nil
}

func (s *Store) GetGrant(_ context.Context, subject, tenantID string) (model.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[grantKey(subject, tenantID)]
	if !ok {
		return model.Grant{}, fmt.Errorf("%w: %s", model.ErrNotFound, grantKey(subject, tenantID))
	}
	return g, nil
}

func (s *Store) UpdateGrant(_ context.Context, g model.Grant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := grantKey(g.Subject, g.TenantID)
	if _, ok := s.grants[key]; !ok {
		return fmt.Errorf("%w: %s", model.ErrNotFound, key)
	}
	s.grants[key] = g
	return nil
}

func (s *Store) ListGrantsBySubject(_ context.Context, subject string) ([]model.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Grant
	for _, g := range s.grants {
		if g.Subject == subject {
			out = append(out, g)
		}
	}
	return out, nil
}

func (s *Store) ListGrantsByTenant(_ context.Context, tenantID string) ([]model.Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Grant
	for _, g := range s.grants {
		if g.TenantID == tenantID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (s *Store) DeleteGrant(_ context.Context, subject, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.grants, grantKey(subject, tenantID))
	return nil
}

func (s *Store) CreatePAT(_ context.Context, p model.PAT) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pats[p.ID]; ok {
		return fmt.Errorf("%w: %s", model.ErrConflict, p.ID)
	}
	if p.Name != "" {
		for _, existing := range s.pats {
			if existing.Subject == p.Subject && existing.Name == p.Name {
				return fmt.Errorf("%w: PAT named %q for %s", model.ErrConflict, p.Name, p.Subject)
			}
		}
	}
	s.pats[p.ID] = p
	return nil
}

func (s *Store) GetPAT(_ context.Context, id string) (model.PAT, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pats[id]
	if !ok {
		return model.PAT{}, fmt.Errorf("%w: %s", model.ErrNotFound, id)
	}
	return p, nil
}

func (s *Store) ListPATsBySubject(_ context.Context, subject string) ([]model.PAT, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.PAT
	for _, p := range s.pats {
		if p.Subject == subject {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Store) DeletePAT(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pats, id)
	return nil
}
