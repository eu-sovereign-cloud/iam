package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// GetUser fetches a single User by subject. Admin-only.
type GetUser struct {
	Users ports.UserStore
}

func (c *GetUser) Do(ctx context.Context, subject string) (model.User, error) {
	if err := model.RequireAdmin(ctx); err != nil {
		return model.User{}, err
	}
	return c.Users.GetUser(ctx, subject)
}
