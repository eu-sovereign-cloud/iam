package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteUser removes a User. Callers are responsible for enforcing that
// only admins invoke it.
type DeleteUser struct {
	Users ports.UserStore
}

func (c *DeleteUser) Do(ctx context.Context, subject string) error {
	return c.Users.DeleteUser(ctx, subject)
}
