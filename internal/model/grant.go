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
}
