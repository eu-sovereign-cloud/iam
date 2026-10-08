package kuberbac

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/pkg/kube"
)

// tenantAdminProviders lists the providers the tenant-admin Role covers.
// ecp compares the permission provider exactly (only resources and verb are
// glob-matched), so "*" is not a wildcard there and every provider must be
// listed explicitly, mirroring the spec's system roles.
var tenantAdminProviders = []string{
	"seca.authorization",
	"seca.region",
	"seca.workspace",
	"seca.network",
	"seca.compute",
	"seca.storage",
}

var tenantAdminPermissions = func() roleSpec {
	perms := make([]permission, 0, len(tenantAdminProviders))
	for _, p := range tenantAdminProviders {
		perms = append(perms, permission{Provider: p, Resources: []string{"*"}, Verb: []string{"*"}})
	}
	return roleSpec{Permissions: perms}
}()

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
// model.TenantAdminRole Role for tenantID with wildcard resources and verbs on every provider.
// Every call re-applies the full desired spec (true "restore" semantics),
// so it's safe to call both at tenant creation and repeatedly from repair.
func (s *Store) EnsureTenantAdminRole(ctx context.Context, tenantID string) error {
	ns := tenantNamespace(tenantID)
	if err := s.ensureNamespace(ctx, ns); err != nil {
		return err
	}
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
