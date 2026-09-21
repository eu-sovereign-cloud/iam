package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// GetUser fetches a single User by subject.
type GetUser struct {
	users ports.UserStore
}

func NewGetUser(users ports.UserStore) *GetUser {
	return &GetUser{users: users}
}

func (c *GetUser) Do(ctx context.Context, subject string) (model.User, error) {
	return c.users.GetUser(ctx, subject)
}
