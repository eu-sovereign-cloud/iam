package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// SetUserAdmin toggles a User's admin flag. Admin-only.
type SetUserAdmin struct {
	Users ports.UserStore
}

func (c *SetUserAdmin) Do(ctx context.Context, subject string, admin bool) (model.User, error) {
	if err := model.RequireAdmin(ctx); err != nil {
		return model.User{}, err
	}
	u, err := c.Users.GetUser(ctx, subject)
	if err != nil {
		return model.User{}, err
	}
	u.Admin = admin
	if err := c.Users.UpdateUser(ctx, u); err != nil {
		return model.User{}, err
	}
	return u, nil
}
