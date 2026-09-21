package ports

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// TenantRoleStore is the persistence port for a tenant's ecp-side RBAC
// state (a Role plus per-subject RoleAssignments) — see doc/adr/0018.
type TenantRoleStore interface {
	// EnsureTenantAdminRole creates or overwrites the canonical
	// model.TenantAdminRole Role for tenantID. Idempotent: every call
	// re-applies the full desired state, so it's safe to call both at
	// tenant creation and repeatedly from a repair.
	EnsureTenantAdminRole(ctx context.Context, tenantID string) error
	DeleteTenantAdminRole(ctx context.Context, tenantID string) error

	// SetRoleAssignment creates or overwrites the RoleAssignment binding
	// subject to roles within tenantID.
	SetRoleAssignment(ctx context.Context, tenantID, subject string, roles []string) error
	DeleteRoleAssignment(ctx context.Context, tenantID, subject string) error

	// ListRoleAssignments lists every IAM-managed RoleAssignment in
	// tenantID.
	ListRoleAssignments(ctx context.Context, tenantID string) ([]model.RoleAssignment, error)
}
