package controller_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// In-memory fakes for the ports controller_test needs. Kept intentionally
// dumb (no k8s involved) so these tests exercise only business logic; the
// real Kubernetes-backed implementation is covered by internal/adapter's
// own tests.

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
	mu   sync.Mutex
	pats map[string]model.PAT
}

func newFakePATStore() *fakePATStore {
	return &fakePATStore{pats: map[string]model.PAT{}}
}

func (f *fakePATStore) CreatePAT(_ context.Context, p model.PAT) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.pats[p.ID]; ok {
		return fmt.Errorf("%w: %s", model.ErrConflict, p.ID)
	}
	if p.Name != "" {
		for _, existing := range f.pats {
			if existing.Subject == p.Subject && existing.Name == p.Name {
				return fmt.Errorf("%w: PAT named %q for %s", model.ErrConflict, p.Name, p.Subject)
			}
		}
	}
	f.pats[p.ID] = p
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
	delete(f.pats, id)
	return nil
}

// fakeSigner round-trips model.Claims through JSON+base64 instead of real
// ES256 signing/verification — enough to exercise CreatePAT/AuthenticatePAT's
// orchestration logic (sign at creation, verify at authentication) without
// needing real crypto in these tests. The real ES256 Sign/Verify path is
// covered by internal/adapter's own tests against the real Signer.
type fakeSigner struct{}

func (fakeSigner) Sign(claims model.Claims) (string, error) {
	raw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	return "fake." + base64.RawURLEncoding.EncodeToString(raw) + ".signed", nil
}

func (fakeSigner) Verify(token string) (model.Claims, error) {
	var claims model.Claims
	body := strings.TrimSuffix(strings.TrimPrefix(token, "fake."), ".signed")
	if body == token {
		return model.Claims{}, fmt.Errorf("not a fake-signed token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return model.Claims{}, err
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return model.Claims{}, err
	}
	// Expiry is deliberately not checked here: AuthenticatePAT's own
	// pat.Expired(clock.Now()) check (against the injected, test-controlled
	// Clock) is what tests rely on for expiry behavior, not wall-clock time.
	return claims, nil
}
