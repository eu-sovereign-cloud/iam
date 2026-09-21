package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// SetUserAdmin toggles a User's admin flag. Callers are responsible for
// enforcing that only admins invoke it.
type SetUserAdmin struct {
	Users ports.UserStore
}

func (c *SetUserAdmin) Do(ctx context.Context, subject string, admin bool) (model.User, error) {
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
