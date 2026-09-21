package kubestore

import (
	"context"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/eu-sovereign-cloud/iam/internal/adapter"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func userToConfigMap(u model.User, namespace string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	cm.Name = userName(u.Subject)
	cm.Namespace = namespace
	cm.Labels = map[string]string{
		labelType:  typeUser,
		labelAdmin: strconv.FormatBool(u.Admin),
	}
	cm.Data = map[string]string{
		dataSubject:     u.Subject,
		dataDisplayName: u.DisplayName,
		dataAdmin:       strconv.FormatBool(u.Admin),
		dataCreatedAt:   u.CreatedAt.Format(time.RFC3339),
	}
	return cm
}

func userFromConfigMap(cm *corev1.ConfigMap) model.User {
	admin, _ := strconv.ParseBool(cm.Data[dataAdmin])
	created, _ := time.Parse(time.RFC3339, cm.Data[dataCreatedAt])
	return model.User{
		Subject:     cm.Data[dataSubject],
		DisplayName: cm.Data[dataDisplayName],
		Admin:       admin,
		CreatedAt:   created,
	}
}

func (s *Store) CreateUser(ctx context.Context, u model.User) error {
	s.mu.RLock()
	_, exists := s.users[u.Subject]
	s.mu.RUnlock()
	if exists {
		return fmt.Errorf("%w: user %q", model.ErrConflict, u.Subject)
	}

	cm := userToConfigMap(u, s.namespace)
	if _, err := s.client.CoreV1().ConfigMaps(s.namespace).Create(ctx, cm, adapter.MetaCreateOpts()); err != nil {
		if adapter.IsAlreadyExists(err) {
			return fmt.Errorf("%w: user %q", model.ErrConflict, u.Subject)
		}
		return fmt.Errorf("creating user %q: %w", u.Subject, err)
	}

	s.mu.Lock()
	s.users[u.Subject] = u
	s.mu.Unlock()
	return nil
}

func (s *Store) GetUser(_ context.Context, subject string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[subject]
	if !ok {
		return model.User{}, fmt.Errorf("%w: user %q", model.ErrNotFound, subject)
	}
	return u, nil
}

func (s *Store) ListUsers(_ context.Context) ([]model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out, nil
}

func (s *Store) UpdateUser(ctx context.Context, u model.User) error {
	s.mu.RLock()
	_, exists := s.users[u.Subject]
	s.mu.RUnlock()
	if !exists {
		return fmt.Errorf("%w: user %q", model.ErrNotFound, u.Subject)
	}

	cm := userToConfigMap(u, s.namespace)
	if _, err := s.client.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, adapter.MetaUpdateOpts()); err != nil {
		return fmt.Errorf("updating user %q: %w", u.Subject, err)
	}

	s.mu.Lock()
	s.users[u.Subject] = u
	s.mu.Unlock()
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, subject string) error {
	name := userName(subject)
	if err := s.client.CoreV1().ConfigMaps(s.namespace).Delete(ctx, name, adapter.MetaDeleteOpts()); err != nil && !adapter.IsNotFound(err) {
		return fmt.Errorf("deleting user %q: %w", subject, err)
	}

	s.mu.Lock()
	delete(s.users, subject)
	s.mu.Unlock()
	return nil
}
