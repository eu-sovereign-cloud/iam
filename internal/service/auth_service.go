package service

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// AuthService resolves a raw bearer credential (an Authorization header
// value or a cookie, depending on the caller) to the User it belongs to.
// Both internal/controller (REST, header-based) and internal/web
// (cookie-based) authenticate through this same use case.
type AuthService struct {
	pats  *PATService
	users UserStore
}

func NewAuthService(pats *PATService, users UserStore) *AuthService {
	return &AuthService{pats: pats, users: users}
}

// Authenticate resolves rawPAT to its owning, still-existing User.
func (s *AuthService) Authenticate(ctx context.Context, rawPAT string) (model.User, error) {
	pat, err := s.pats.Authenticate(ctx, rawPAT)
	if err != nil {
		return model.User{}, err
	}
	return s.users.GetUser(ctx, pat.Subject)
}
