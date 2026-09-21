package model

// RoleAssignment is IAM's own minimal view of a subject's ecp role
// binding within a tenant — deliberately not shaped like ecp's
// RoleAssignment CRD (Subs/Scopes/Roles, one object can bind many
// subjects to many roles across many tenants): only
// internal/adapter/kuberbac needs to know that shape. See doc/adr/0018.
type RoleAssignment struct {
	Subject  string
	TenantID string
	Roles    []string
}
