package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListUsers lists all registered Users.
type ListUsers struct {
	Users ports.UserStore
}

func (c *ListUsers) Do(ctx context.Context) ([]model.User, error) {
	return c.Users.ListUsers(ctx)
}
