package controller

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
// identityFromContext.
func (c *Controller) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r)
		if raw == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		user, err := c.Auth.Authenticate(r.Context(), raw)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or revoked token")
			return
		}
		next(w, r.WithContext(withIdentity(r.Context(), user)))
	}
}

// RequireAdmin additionally requires the authenticated caller to have the
// admin flag set (ADR 0008: only admins manage Tenants/Users/Grants).
func (c *Controller) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return c.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		if !identityFromContext(r.Context()).Admin {
			writeError(w, http.StatusForbidden, "admin privileges required")
			return
		}
		next(w, r)
	})
}

// RequireSelfOrAdmin requires the authenticated caller to either be an
// admin or to be acting on their own subject, as given by the {subject}
// path value (self-service PAT management, ADR 0008).
func (c *Controller) RequireSelfOrAdmin(next http.HandlerFunc) http.HandlerFunc {
	return c.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		caller := identityFromContext(r.Context())
		subject := r.PathValue("subject")
		if !caller.Admin && caller.Subject != subject {
			writeError(w, http.StatusForbidden, "may only manage your own resources")
			return
		}
		next(w, r)
	})
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
