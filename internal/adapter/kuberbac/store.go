// Package kuberbac talks directly to ecp's Role/RoleAssignment custom
// resources (group authorization.v1.secapi.cloud/v1) as a Kubernetes
// backchannel — the same cluster ecp runs in — rather than through ecp's
// REST/HTTP frontend or its external go-sdk. See doc/adr/0018 for why:
// ecp has no Tenant CRD (a tenant is just hex(sha3-224(tenantID)) as a
// namespace, auto-provisioned by ecp on first write), and ecp's own Go
// types for these CRDs live in a multi-module repo whose nested-module
// tags don't actually resolve outside a go.work checkout.
package kuberbac

import "k8s.io/client-go/dynamic"

// Store implements ports.TenantRoleStore against ecp's Role/RoleAssignment
// CRDs. Unlike kubestore, it holds no in-memory cache and touches no
// namespace of its own — every call addresses a tenant's already
// (auto-)provisioned namespace directly.
type Store struct {
	dynamic dynamic.Interface
}

func New(dynamic dynamic.Interface) *Store {
	return &Store{dynamic: dynamic}
}
