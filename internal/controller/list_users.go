package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListUsers lists all registered Users. Admin-only.
type ListUsers struct {
	Users ports.UserStore
}

func (c *ListUsers) Do(ctx context.Context) ([]model.User, error) {
	if err := model.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	return c.Users.ListUsers(ctx)
}
