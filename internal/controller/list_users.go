package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListUsers lists all registered Users.
type ListUsers struct {
	users ports.UserStore
}

func NewListUsers(users ports.UserStore) *ListUsers {
	return &ListUsers{users: users}
}

func (c *ListUsers) Do(ctx context.Context) ([]model.User, error) {
	return c.users.ListUsers(ctx)
}
