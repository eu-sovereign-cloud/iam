package kuberbac

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/pkg/kube"
)

var tenantAdminPermissions = roleSpec{
	Permissions: []permission{{Provider: "*", Resources: []string{"*"}, Verb: []string{"*"}}},
}

func roleObject(tenantID string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(group + "/" + version)
	obj.SetKind("Role")
	obj.SetName(model.TenantAdminRole)
	obj.SetNamespace(tenantNamespace(tenantID))
	if err := setSpec(obj, tenantAdminPermissions); err != nil {
		return nil, fmt.Errorf("encoding tenant-admin Role spec: %w", err)
	}
	return obj, nil
}

// EnsureTenantAdminRole creates or overwrites the canonical
// model.TenantAdminRole Role for tenantID with wildcard permissions.
// Every call re-applies the full desired spec (true "restore" semantics),
// so it's safe to call both at tenant creation and repeatedly from repair.
func (s *Store) EnsureTenantAdminRole(ctx context.Context, tenantID string) error {
	ns := tenantNamespace(tenantID)
	obj, err := roleObject(tenantID)
	if err != nil {
		return err
	}

	client := s.dynamic.Resource(roleGVR).Namespace(ns)
	existing, err := client.Get(ctx, model.TenantAdminRole, kube.MetaGetOpts())
	if err != nil {
		if !kube.IsNotFound(err) {
			return fmt.Errorf("getting tenant-admin Role for tenant %q: %w", tenantID, err)
		}
		if _, err := client.Create(ctx, obj, kube.MetaCreateOpts()); err != nil {
			return fmt.Errorf("creating tenant-admin Role for tenant %q: %w", tenantID, err)
		}
		return nil
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	if _, err := client.Update(ctx, obj, kube.MetaUpdateOpts()); err != nil {
		return fmt.Errorf("updating tenant-admin Role for tenant %q: %w", tenantID, err)
	}
	return nil
}

func (s *Store) DeleteTenantAdminRole(ctx context.Context, tenantID string) error {
	ns := tenantNamespace(tenantID)
	if err := s.dynamic.Resource(roleGVR).Namespace(ns).Delete(ctx, model.TenantAdminRole, kube.MetaDeleteOpts()); err != nil && !kube.IsNotFound(err) {
		return fmt.Errorf("deleting tenant-admin Role for tenant %q: %w", tenantID, err)
	}
	return nil
}
