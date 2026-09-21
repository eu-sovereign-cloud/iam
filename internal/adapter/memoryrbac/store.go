// Package memoryrbac is a non-persistent, in-memory implementation of
// ports.TenantRoleStore, for tests that need a real implementation of
// that port without a Kubernetes dynamic client. See kuberbac for the
// production, ecp-Role/RoleAssignment-CRD-backed implementation this
// exists alongside.
package memoryrbac

import (
	"context"
	"fmt"
	"sync"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type roleAssignmentKey struct{ tenantID, subject string }

type Store struct {
	mu sync.Mutex

	tenantAdminRoles map[string]bool // tenantID -> role exists
	assignments      map[roleAssignmentKey][]string
}

func New() *Store {
	return &Store{
		tenantAdminRoles: map[string]bool{},
		assignments:      map[roleAssignmentKey][]string{},
	}
}

func (s *Store) EnsureTenantAdminRole(_ context.Context, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenantAdminRoles[tenantID] = true
	return nil
}

func (s *Store) DeleteTenantAdminRole(_ context.Context, tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tenantAdminRoles, tenantID)
	return nil
}

// HasTenantAdminRole is a test-only accessor letting assertions check
// EnsureTenantAdminRole/DeleteTenantAdminRole were actually called.
func (s *Store) HasTenantAdminRole(tenantID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tenantAdminRoles[tenantID]
}

func (s *Store) SetRoleAssignment(_ context.Context, tenantID, subject string, roles []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assignments[roleAssignmentKey{tenantID, subject}] = roles
	return nil
}

func (s *Store) DeleteRoleAssignment(_ context.Context, tenantID, subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.assignments, roleAssignmentKey{tenantID, subject})
	return nil
}

func (s *Store) ListRoleAssignments(_ context.Context, tenantID string) ([]model.RoleAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.RoleAssignment
	for key, roles := range s.assignments {
		if key.tenantID == tenantID {
			out = append(out, model.RoleAssignment{Subject: key.subject, TenantID: tenantID, Roles: roles})
		}
	}
	return out, nil
}

// RoleAssignment is a test-only accessor for a single binding, so
// assertions can check the exact roles a subject ended up bound to
// without going through ListRoleAssignments.
func (s *Store) RoleAssignment(tenantID, subject string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	roles, ok := s.assignments[roleAssignmentKey{tenantID, subject}]
	if !ok {
		return nil, fmt.Errorf("no RoleAssignment for %s/%s", tenantID, subject)
	}
	return roles, nil
}
