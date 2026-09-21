package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateUser registers a User. Admin-only.
type CreateUser struct {
	Users ports.UserStore
	Clock ports.Clock
}

func (c *CreateUser) Do(ctx context.Context, subject, displayName string, admin bool) (model.User, error) {
	if err := model.RequireAdmin(ctx); err != nil {
		return model.User{}, err
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return model.User{}, fmt.Errorf("%w: subject is required", model.ErrInvalid)
	}
	u := model.User{
		Subject:     subject,
		DisplayName: strings.TrimSpace(displayName),
		Admin:       admin,
		CreatedAt:   c.Clock.Now(),
	}
	if err := c.Users.CreateUser(ctx, u); err != nil {
		return model.User{}, err
	}
	return u, nil
}
