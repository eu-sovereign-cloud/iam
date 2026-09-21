package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// AuthenticateUser resolves a raw bearer credential (an Authorization
// header value or a cookie, depending on the caller) to the User it
// belongs to. Both internal/service (REST, header-based) and internal/web
// (cookie-based) authenticate through this same use case.
type AuthenticateUser struct {
	pats  *AuthenticatePAT
	users ports.UserStore
}

func NewAuthenticateUser(pats *AuthenticatePAT, users ports.UserStore) *AuthenticateUser {
	return &AuthenticateUser{pats: pats, users: users}
}

// Do resolves rawPAT to its owning, still-existing User.
func (c *AuthenticateUser) Do(ctx context.Context, rawPAT string) (model.User, error) {
	pat, err := c.pats.Do(ctx, rawPAT)
	if err != nil {
		return model.User{}, err
	}
	return c.users.GetUser(ctx, pat.Subject)
}
