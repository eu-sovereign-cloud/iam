package kuberbac

import (
	"context"
	"crypto/sha3"
	"encoding/hex"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/iam/internal/pkg/kube"
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

var namespaceGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}

// ensureNamespace creates the tenant's namespace if it doesn't already
// exist. ecp's own docs describe this namespace as "auto-provisioned"
// the first time a tenant-scoped object is written — but that
// provisioning is ecp's own REST write-path logic (doc/adr/0018), which
// this backchannel bypasses entirely. Without this, writing a Role/
// RoleAssignment straight to the Kubernetes API into a namespace nothing
// has created yet fails outright (confirmed by a real e2e run against a
// kind cluster with no ecp deployed in it).
func (s *Store) ensureNamespace(ctx context.Context, ns string) error {
	client := s.dynamic.Resource(namespaceGVR)
	if _, err := client.Get(ctx, ns, kube.MetaGetOpts()); err == nil {
		return nil
	} else if !kube.IsNotFound(err) {
		return fmt.Errorf("getting namespace %q: %w", ns, err)
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("Namespace")
	obj.SetName(ns)
	if _, err := client.Create(ctx, obj, kube.MetaCreateOpts()); err != nil && !kube.IsAlreadyExists(err) {
		return fmt.Errorf("creating namespace %q: %w", ns, err)
	}
	return nil
}
