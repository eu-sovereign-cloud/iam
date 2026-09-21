package kuberbac

import (
	"crypto/sha3"
	"encoding/hex"
)

// tenantNamespace reimplements ecp's own tenant-namespace hashing
// (ecp/framework/backend/kubernetes/adapter.go's ComputeNamespace, for
// the tenant-only / no-workspace case: hex(SHA3-224(tenantID))). IAM and
// ecp must compute byte-for-byte the same value — it's how ecp itself
// decides which Kubernetes namespace a tenant's Role/RoleAssignment
// objects live in, and there is no API to ask ecp for it directly.
func tenantNamespace(tenantID string) string {
	sum := sha3.Sum224([]byte(tenantID))
	return hex.EncodeToString(sum[:])
}
