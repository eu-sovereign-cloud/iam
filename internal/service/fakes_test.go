package service_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// In-memory fakes for the ports service_test needs. Kept intentionally
// dumb (no k8s involved) so service-layer tests exercise only business
// logic; the real Kubernetes-backed implementation is covered by
// internal/adapter's own tests.

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeUserStore struct {
	mu    sync.Mutex
	users map[string]model.User
}

func newFakeUserStore() *fakeUserStore { return &fakeUserStore{users: map[string]model.User{}} }

func (f *fakeUserStore) CreateUser(_ context.Context, u model.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[u.Subject]; ok {
		return fmt.Errorf("%w: %s", model.ErrConflict, u.Subject)
	}
	f.users[u.Subject] = u
	return nil
}

func (f *fakeUserStore) GetUser(_ context.Context, subject string) (model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[subject]
	if !ok {
		return model.User{}, fmt.Errorf("%w: %s", model.ErrNotFound, subject)
	}
	return u, nil
}

func (f *fakeUserStore) ListUsers(_ context.Context) ([]model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]model.User, 0, len(f.users))
	for _, u := range f.users {
		out = append(out, u)
	}
	return out, nil
}

func (f *fakeUserStore) UpdateUser(_ context.Context, u model.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[u.Subject]; !ok {
		return fmt.Errorf("%w: %s", model.ErrNotFound, u.Subject)
	}
	f.users[u.Subject] = u
	return nil
}

func (f *fakeUserStore) DeleteUser(_ context.Context, subject string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.users, subject)
	return nil
}

type fakeTenantStore struct {
	mu      sync.Mutex
	tenants map[string]model.Tenant
}

func newFakeTenantStore() *fakeTenantStore {
	return &fakeTenantStore{tenants: map[string]model.Tenant{}}
}

func (f *fakeTenantStore) CreateTenant(_ context.Context, t model.Tenant) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.tenants[t.TenantID]; ok {
		return fmt.Errorf("%w: %s", model.ErrConflict, t.TenantID)
	}
	f.tenants[t.TenantID] = t
	return nil
}

func (f *fakeTenantStore) GetTenant(_ context.Context, tenantID string) (model.Tenant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tenants[tenantID]
	if !ok {
		return model.Tenant{}, fmt.Errorf("%w: %s", model.ErrNotFound, tenantID)
	}
	return t, nil
}

func (f *fakeTenantStore) ListTenants(_ context.Context) ([]model.Tenant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]model.Tenant, 0, len(f.tenants))
	for _, t := range f.tenants {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeTenantStore) DeleteTenant(_ context.Context, tenantID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.tenants, tenantID)
	return nil
}

type fakeGrantStore struct {
	mu     sync.Mutex
	grants map[string]model.Grant
}

func newFakeGrantStore() *fakeGrantStore { return &fakeGrantStore{grants: map[string]model.Grant{}} }

func grantKey(subject, tenantID string) string { return subject + "|" + tenantID }

func (f *fakeGrantStore) CreateGrant(_ context.Context, g model.Grant) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := grantKey(g.Subject, g.TenantID)
	if _, ok := f.grants[key]; ok {
		return fmt.Errorf("%w: %s", model.ErrConflict, key)
	}
	f.grants[key] = g
	return nil
}

func (f *fakeGrantStore) ListGrantsBySubject(_ context.Context, subject string) ([]model.Grant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []model.Grant
	for _, g := range f.grants {
		if g.Subject == subject {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeGrantStore) DeleteGrant(_ context.Context, subject, tenantID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.grants, grantKey(subject, tenantID))
	return nil
}

type fakePATStore struct {
	mu         sync.Mutex
	pats       map[string]model.PAT
	patsByHash map[string]string
}

func newFakePATStore() *fakePATStore {
	return &fakePATStore{pats: map[string]model.PAT{}, patsByHash: map[string]string{}}
}

func (f *fakePATStore) CreatePAT(_ context.Context, p model.PAT) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pats[p.ID] = p
	f.patsByHash[p.TokenHash] = p.ID
	return nil
}

func (f *fakePATStore) GetPAT(_ context.Context, id string) (model.PAT, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.pats[id]
	if !ok {
		return model.PAT{}, fmt.Errorf("%w: %s", model.ErrNotFound, id)
	}
	return p, nil
}

func (f *fakePATStore) GetPATByHash(_ context.Context, hash string) (model.PAT, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.patsByHash[hash]
	if !ok {
		return model.PAT{}, fmt.Errorf("%w: PAT", model.ErrNotFound)
	}
	return f.pats[id], nil
}

func (f *fakePATStore) ListPATsBySubject(_ context.Context, subject string) ([]model.PAT, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []model.PAT
	for _, p := range f.pats {
		if p.Subject == subject {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakePATStore) DeletePAT(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.pats[id]; ok {
		delete(f.patsByHash, p.TokenHash)
	}
	delete(f.pats, id)
	return nil
}

// fakeTokenGenerator returns deterministic, easily-inspectable tokens
// instead of random ones, and hashes with a trivial prefix rather than
// SHA-256 (adapter.TokenGenerator's real hashing is covered separately).
type fakeTokenGenerator struct{ counter int }

func (f *fakeTokenGenerator) NewToken() (raw string, hash string) {
	f.counter++
	raw = fmt.Sprintf("raw-token-%d", f.counter)
	return raw, f.Hash(raw)
}

func (f *fakeTokenGenerator) Hash(raw string) string { return "hash:" + raw }

type fakeSigner struct{}

func (fakeSigner) Sign(claims model.Claims) (string, error) {
	return fmt.Sprintf("signed(sub=%s,tenants=%v)", claims.Subject, claims.Tenants), nil
}
