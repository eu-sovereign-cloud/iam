package model

import "time"

// Grant records that Subject is meant to have access to TenantID. It is
// IAM-internal bookkeeping only: creating a Grant does not by itself create
// any RBAC permission inside ecp (see ADR 0008 for the deferred integration
// point). A Subject's current Grants are what gets resolved into the
// "tenants" claim of a JWT at token-exchange time.
type Grant struct {
	Subject   string
	TenantID  string
	GrantedAt time.Time
	GrantedBy string
	// Admin marks Subject as a tenant admin for TenantID: they may grant
	// or revoke other subjects' access to this one tenant, but hold no
	// privilege over any other tenant and cannot themselves promote or
	// demote tenant-admin status (only a global admin can, via
	// SetGrantAdmin — see doc/adr/0016).
	Admin bool
	// Roles are the ecp Role names Subject is bound to within TenantID via
	// a RoleAssignment (see doc/adr/0018) — a RoleAssignment may grant
	// more than one role at once. At least one is required. While Admin
	// is true, the subject is actually bound to only TenantAdminRole
	// instead — Roles is still stored so demoting reverts the binding to
	// it.
	Roles []string
}
