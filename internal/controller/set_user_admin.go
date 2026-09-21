package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// SetUserAdmin toggles a User's admin flag. Callers are responsible for
// enforcing that only admins invoke it.
type SetUserAdmin struct {
	users ports.UserStore
}

func NewSetUserAdmin(users ports.UserStore) *SetUserAdmin {
	return &SetUserAdmin{users: users}
}

func (c *SetUserAdmin) Do(ctx context.Context, subject string, admin bool) (model.User, error) {
	u, err := c.users.GetUser(ctx, subject)
	if err != nil {
		return model.User{}, err
	}
	u.Admin = admin
	if err := c.users.UpdateUser(ctx, u); err != nil {
		return model.User{}, err
	}
	return u, nil
}
