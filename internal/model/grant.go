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
}
