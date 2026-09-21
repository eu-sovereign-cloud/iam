package ports

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// UserStore is the persistence port for Users.
type UserStore interface {
	CreateUser(ctx context.Context, u model.User) error
	GetUser(ctx context.Context, subject string) (model.User, error)
	ListUsers(ctx context.Context) ([]model.User, error)
	UpdateUser(ctx context.Context, u model.User) error
	DeleteUser(ctx context.Context, subject string) error
}
