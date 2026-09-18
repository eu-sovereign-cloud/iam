package web

import (
	"context"
	"net/http"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type identityContextKey struct{}

func withIdentity(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, identityContextKey{}, u)
}

func identityFromContext(ctx context.Context) model.User {
	u, _ := ctx.Value(identityContextKey{}).(model.User)
	return u
}

func (wb *Web) authenticateRequest(r *http.Request) (model.User, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return model.User{}, false
	}
	user, err := wb.Auth.Authenticate(r.Context(), cookie.Value)
	if err != nil {
		return model.User{}, false
	}
	return user, true
}

func (wb *Web) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := wb.authenticateRequest(r)
		if !ok {
			http.Redirect(w, r, "/web/login", http.StatusSeeOther)
			return
		}
		next(w, r.WithContext(withIdentity(r.Context(), user)))
	}
}

func (wb *Web) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return wb.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if !identityFromContext(r.Context()).Admin {
			http.Error(w, "admin privileges required", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func (wb *Web) handleRoot(w http.ResponseWriter, r *http.Request) {
	if _, ok := wb.authenticateRequest(r); ok {
		http.Redirect(w, r, "/web/pats", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/web/login", http.StatusSeeOther)
}

func (wb *Web) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	wb.render(w, "login.html", map[string]any{})
}

func (wb *Web) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		wb.render(w, "login.html", map[string]any{"Error": "invalid form submission"})
		return
	}
	pat := r.FormValue("pat")
	if _, err := wb.Auth.Authenticate(r.Context(), pat); err != nil {
		wb.render(w, "login.html", map[string]any{"Error": "invalid or revoked token"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    pat,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/web/pats", http.StatusSeeOther)
}

func (wb *Web) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/web/login", http.StatusSeeOther)
}
