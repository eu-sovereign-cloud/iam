package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// DeleteUser removes a User. Admin-only.
type DeleteUser struct {
	Users ports.UserStore
}

func (c *DeleteUser) Do(ctx context.Context, subject string) error {
	if err := model.RequireAdmin(ctx); err != nil {
		return err
	}
	return c.Users.DeleteUser(ctx, subject)
}
