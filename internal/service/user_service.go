package service

import (
	"context"
	"fmt"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// UserService manages Users. Callers (the controller layer) are
// responsible for enforcing that only admins invoke the mutating methods.
type UserService struct {
	store UserStore
	clock Clock
}

func NewUserService(store UserStore, clock Clock) *UserService {
	return &UserService{store: store, clock: clock}
}

func (s *UserService) Create(ctx context.Context, subject, displayName string, admin bool) (model.User, error) {
	if subject == "" {
		return model.User{}, fmt.Errorf("%w: subject is required", model.ErrInvalid)
	}
	u := model.User{
		Subject:     subject,
		DisplayName: displayName,
		Admin:       admin,
		CreatedAt:   s.clock.Now(),
	}
	if err := s.store.CreateUser(ctx, u); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func (s *UserService) Get(ctx context.Context, subject string) (model.User, error) {
	return s.store.GetUser(ctx, subject)
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	return s.store.ListUsers(ctx)
}

func (s *UserService) SetAdmin(ctx context.Context, subject string, admin bool) (model.User, error) {
	u, err := s.store.GetUser(ctx, subject)
	if err != nil {
		return model.User{}, err
	}
	u.Admin = admin
	if err := s.store.UpdateUser(ctx, u); err != nil {
		return model.User{}, err
	}
	return u, nil
}

func (s *UserService) Delete(ctx context.Context, subject string) error {
	return s.store.DeleteUser(ctx, subject)
}
