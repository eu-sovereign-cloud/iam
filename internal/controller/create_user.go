package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// CreateUser registers a User. Callers are responsible for enforcing that
// only admins invoke it.
type CreateUser struct {
	Users ports.UserStore
	Clock ports.Clock
}

func (c *CreateUser) Do(ctx context.Context, subject, displayName string, admin bool) (model.User, error) {
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
