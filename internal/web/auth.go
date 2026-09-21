package web

import (
	"net/http"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func (wb *Web) authenticateRequest(r *http.Request) (model.User, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return model.User{}, false
	}
	user, err := wb.AuthenticateUser.Do(r.Context(), cookie.Value)
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
		next(w, r.WithContext(model.WithIdentity(r.Context(), user)))
	}
}

func (wb *Web) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return wb.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if !model.IdentityFromContext(r.Context()).Admin {
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
		wb.render(w, "login.html", map[string]any{"Error": "That submission did not come through. Try again."})
		return
	}
	pat := r.FormValue("pat")
	if _, err := wb.AuthenticateUser.Do(r.Context(), pat); err != nil {
		wb.render(w, "login.html", map[string]any{"Error": "That token is not valid, or it has been revoked."})
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
