package kuberbac_test

import (
	"context"
	"crypto/sha3"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/eu-sovereign-cloud/iam/internal/adapter/kuberbac"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

var (
	roleGVR           = schema.GroupVersionResource{Group: "authorization.v1.secapi.cloud", Version: "v1", Resource: "roles"}
	roleAssignmentGVR = schema.GroupVersionResource{Group: "authorization.v1.secapi.cloud", Version: "v1", Resource: "role-assignments"}
)

func newTestStore() (*kuberbac.Store, *dynamicfake.FakeDynamicClient) {
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		roleGVR:           "RoleList",
		roleAssignmentGVR: "RoleAssignmentList",
	})
	return kuberbac.New(client), client
}

// tenantNamespace independently computes ecp's own tenant-namespace hash
// (hex(SHA3-224(tenantID))) using the same stdlib primitive kuberbac
// uses internally, so these tests confirm objects land in the namespace
// ecp itself would look for them in, without depending on ecp being
// checked out.
func tenantNamespace(tenantID string) string {
	sum := sha3.Sum224([]byte(tenantID))
	return hex.EncodeToString(sum[:])
}

func TestEnsureTenantAdminRole(t *testing.T) {
	ctx := context.Background()
	store, client := newTestStore()
	ns := tenantNamespace("tenant-1")

	require.NoError(t, store.EnsureTenantAdminRole(ctx, "tenant-1"))

	obj, err := client.Resource(roleGVR).Namespace(ns).Get(ctx, model.TenantAdminRole, metav1.GetOptions{})
	require.NoError(t, err)
	perms, found := unstructuredSlice(obj.Object, "spec", "permissions")
	require.True(t, found)
	require.Len(t, perms, 1)

	// Calling again must overwrite, not conflict — repair relies on this.
	require.NoError(t, store.EnsureTenantAdminRole(ctx, "tenant-1"))
	_, err = client.Resource(roleGVR).Namespace(ns).Get(ctx, model.TenantAdminRole, metav1.GetOptions{})
	require.NoError(t, err)
}

func TestDeleteTenantAdminRole(t *testing.T) {
	ctx := context.Background()
	store, client := newTestStore()
	ns := tenantNamespace("tenant-1")

	require.NoError(t, store.EnsureTenantAdminRole(ctx, "tenant-1"))
	require.NoError(t, store.DeleteTenantAdminRole(ctx, "tenant-1"))

	_, err := client.Resource(roleGVR).Namespace(ns).Get(ctx, model.TenantAdminRole, metav1.GetOptions{})
	require.Error(t, err)

	// Deleting a Role that's already gone must not error.
	require.NoError(t, store.DeleteTenantAdminRole(ctx, "tenant-1"))
}

func TestSetRoleAssignment_CreatesAndOverwrites(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore()

	require.NoError(t, store.SetRoleAssignment(ctx, "tenant-1", "alice", []string{"member", "viewer"}))

	got, err := store.ListRoleAssignments(ctx, "tenant-1")
	require.NoError(t, err)
	require.Equal(t, []model.RoleAssignment{{Subject: "alice", TenantID: "tenant-1", Roles: []string{"member", "viewer"}}}, got)

	// Overwrite with a different set of roles (e.g. promoted to tenant-admin).
	require.NoError(t, store.SetRoleAssignment(ctx, "tenant-1", "alice", []string{model.TenantAdminRole}))
	got, err = store.ListRoleAssignments(ctx, "tenant-1")
	require.NoError(t, err)
	require.Equal(t, []model.RoleAssignment{{Subject: "alice", TenantID: "tenant-1", Roles: []string{model.TenantAdminRole}}}, got)
}

func TestDeleteRoleAssignment(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore()

	require.NoError(t, store.SetRoleAssignment(ctx, "tenant-1", "alice", []string{"member"}))
	require.NoError(t, store.DeleteRoleAssignment(ctx, "tenant-1", "alice"))

	got, err := store.ListRoleAssignments(ctx, "tenant-1")
	require.NoError(t, err)
	require.Empty(t, got)

	// Deleting one that's already gone must not error.
	require.NoError(t, store.DeleteRoleAssignment(ctx, "tenant-1", "alice"))
}

func TestListRoleAssignments_ScopedToTenant(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore()

	require.NoError(t, store.SetRoleAssignment(ctx, "tenant-1", "alice", []string{"member"}))
	require.NoError(t, store.SetRoleAssignment(ctx, "tenant-2", "bob", []string{"member"}))

	got, err := store.ListRoleAssignments(ctx, "tenant-1")
	require.NoError(t, err)
	require.Equal(t, []model.RoleAssignment{{Subject: "alice", TenantID: "tenant-1", Roles: []string{"member"}}}, got)
}

func unstructuredSlice(obj map[string]any, fields ...string) ([]any, bool) {
	cur := any(obj)
	for _, f := range fields {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[f]
		if !ok {
			return nil, false
		}
	}
	s, ok := cur.([]any)
	return s, ok
}
