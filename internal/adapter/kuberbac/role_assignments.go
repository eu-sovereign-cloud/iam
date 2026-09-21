package kuberbac

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/pkg/kube"
)

func roleAssignmentObject(tenantID, subject string, roles []string) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(group + "/" + version)
	obj.SetKind("RoleAssignment")
	obj.SetName(roleAssignmentName(subject))
	obj.SetNamespace(tenantNamespace(tenantID))
	obj.SetLabels(map[string]string{
		labelManagedBy: managedByGrant,
		labelSubject:   shortHash(subject),
	})
	spec := roleAssignmentSpec{
		Subs:   []string{subject},
		Roles:  roles,
		Scopes: []roleAssignmentScope{{Tenants: []string{tenantID}}},
	}
	if err := setSpec(obj, spec); err != nil {
		return nil, fmt.Errorf("encoding RoleAssignment spec for %s/%s: %w", tenantID, subject, err)
	}
	return obj, nil
}

// SetRoleAssignment creates or overwrites the RoleAssignment binding
// subject to roles within tenantID. Every call re-applies the full
// desired spec, so it's safe to call repeatedly (promotion/demotion,
// repair).
func (s *Store) SetRoleAssignment(ctx context.Context, tenantID, subject string, roles []string) error {
	ns := tenantNamespace(tenantID)
	if err := s.ensureNamespace(ctx, ns); err != nil {
		return err
	}
	name := roleAssignmentName(subject)
	obj, err := roleAssignmentObject(tenantID, subject, roles)
	if err != nil {
		return err
	}

	client := s.dynamic.Resource(roleAssignmentGVR).Namespace(ns)
	existing, err := client.Get(ctx, name, kube.MetaGetOpts())
	if err != nil {
		if !kube.IsNotFound(err) {
			return fmt.Errorf("getting RoleAssignment for %s/%s: %w", tenantID, subject, err)
		}
		if _, err := client.Create(ctx, obj, kube.MetaCreateOpts()); err != nil {
			return fmt.Errorf("creating RoleAssignment for %s/%s: %w", tenantID, subject, err)
		}
		return nil
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	if _, err := client.Update(ctx, obj, kube.MetaUpdateOpts()); err != nil {
		return fmt.Errorf("updating RoleAssignment for %s/%s: %w", tenantID, subject, err)
	}
	return nil
}

func (s *Store) DeleteRoleAssignment(ctx context.Context, tenantID, subject string) error {
	ns := tenantNamespace(tenantID)
	name := roleAssignmentName(subject)
	if err := s.dynamic.Resource(roleAssignmentGVR).Namespace(ns).Delete(ctx, name, kube.MetaDeleteOpts()); err != nil && !kube.IsNotFound(err) {
		return fmt.Errorf("deleting RoleAssignment for %s/%s: %w", tenantID, subject, err)
	}
	return nil
}

func (s *Store) ListRoleAssignments(ctx context.Context, tenantID string) ([]model.RoleAssignment, error) {
	ns := tenantNamespace(tenantID)
	list, err := s.dynamic.Resource(roleAssignmentGVR).Namespace(ns).List(ctx, kube.MetaListOpts(labelManagedBy+"="+managedByGrant))
	if err != nil {
		return nil, fmt.Errorf("listing RoleAssignments for tenant %q: %w", tenantID, err)
	}

	out := make([]model.RoleAssignment, 0, len(list.Items))
	for i := range list.Items {
		var spec roleAssignmentSpec
		if err := getSpec(&list.Items[i], &spec); err != nil {
			return nil, fmt.Errorf("decoding RoleAssignment %s in tenant %q: %w", list.Items[i].GetName(), tenantID, err)
		}
		if len(spec.Subs) == 0 || len(spec.Roles) == 0 {
			continue
		}
		out = append(out, model.RoleAssignment{Subject: spec.Subs[0], TenantID: tenantID, Roles: spec.Roles})
	}
	return out, nil
}
