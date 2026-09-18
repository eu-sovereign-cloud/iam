// Package adapter implements the driven adapters IAM's services depend on:
// a Kubernetes ConfigMap/Secret-backed store (ADR 0001, ADR 0009) and an
// ES256 JWT signer/verifier (ADR 0005, ADR 0012).
package adapter

import (
	"context"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// Store is an in-memory-cached, write-through Kubernetes store for Users,
// Tenants, Grants and PATs. Per ADR 0009 it loads its full state once via
// Load and never watches for external changes; every mutating method
// writes to the Kubernetes API first and only updates the cache once that
// write succeeds. It is safe only for a single iamd replica.
type Store struct {
	client    kubernetes.Interface
	namespace string

	mu      sync.RWMutex
	users   map[string]model.User   // key: subject
	tenants map[string]model.Tenant // key: tenantID
	grants  map[string]model.Grant  // key: subject + "|" + tenantID
	pats    map[string]model.PAT    // key: id (the PAT JWT's jti)
}

func NewStore(client kubernetes.Interface, namespace string) *Store {
	return &Store{
		client:    client,
		namespace: namespace,
		users:     map[string]model.User{},
		tenants:   map[string]model.Tenant{},
		grants:    map[string]model.Grant{},
		pats:      map[string]model.PAT{},
	}
}

// Load performs one LIST per resource kind and populates the in-memory
// cache. It must be called once before the store is used, after the
// namespace has been ensured to exist.
func (s *Store) Load(ctx context.Context) error {
	if err := s.ensureNamespace(ctx); err != nil {
		return err
	}

	cms, err := s.client.CoreV1().ConfigMaps(s.namespace).List(ctx, metaListOpts(labelType))
	if err != nil {
		return fmt.Errorf("listing configmaps: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, cm := range cms.Items {
		switch cm.Labels[labelType] {
		case typeUser:
			u := userFromConfigMap(&cm)
			s.users[u.Subject] = u
		case typeTenant:
			t := tenantFromConfigMap(&cm)
			s.tenants[t.TenantID] = t
		case typeGrant:
			g := grantFromConfigMap(&cm)
			s.grants[grantKey(g.Subject, g.TenantID)] = g
		case typePAT:
			p, err := patFromConfigMap(&cm)
			if err != nil {
				return fmt.Errorf("decoding PAT configmap %s: %w", cm.Name, err)
			}
			s.pats[p.ID] = p
		}
	}
	return nil
}

func (s *Store) ensureNamespace(ctx context.Context) error {
	_, err := s.client.CoreV1().Namespaces().Get(ctx, s.namespace, metaGetOpts())
	if err == nil {
		return nil
	}
	if !isNotFound(err) {
		return fmt.Errorf("getting namespace %s: %w", s.namespace, err)
	}
	ns := &corev1.Namespace{}
	ns.Name = s.namespace
	if _, err := s.client.CoreV1().Namespaces().Create(ctx, ns, metaCreateOpts()); err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("creating namespace %s: %w", s.namespace, err)
	}
	return nil
}

func grantKey(subject, tenantID string) string {
	return subject + "|" + tenantID
}
