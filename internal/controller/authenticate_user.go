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
	PATs  *AuthenticatePAT
	Users ports.UserStore
}

// Do resolves rawPAT to its owning, still-existing User.
func (c *AuthenticateUser) Do(ctx context.Context, rawPAT string) (model.User, error) {
	pat, err := c.PATs.Do(ctx, rawPAT)
	if err != nil {
		return model.User{}, err
	}
	return c.Users.GetUser(ctx, pat.Subject)
}
