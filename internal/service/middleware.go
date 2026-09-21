package service

import (
	"errors"
	"net/http"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
}

// RequireAuth authenticates the request's Authorization: Bearer <PAT>
// header and, on success, makes the caller's User available via
// model.IdentityFromContext.
func (s *Service) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r)
		if raw == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		user, err := s.AuthenticateUser.Do(r.Context(), raw)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or revoked token")
			return
		}
		next(w, r.WithContext(model.WithIdentity(r.Context(), user)))
	}
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, model.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, model.ErrInvalid):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
