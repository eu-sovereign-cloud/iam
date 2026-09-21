// Package web implements IAM's minimal server-rendered admin/self-service
// UI: plain HTML templates, one small stylesheet, no client-side framework
// (see ADR 0002 and the design plan's low-tech UI constraint).
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/eu-sovereign-cloud/iam/internal/service"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

const cookieName = "iam_pat"

// Web wires the HTML handlers to the service layer. Authentication is
// cookie-based: the cookie value *is* the bearer PAT (ADR: no server-side
// session store, consistent with the k8s-only storage constraint).
type Web struct {
	Auth    *service.AuthService
	Users   *service.UserService
	Tenants *service.TenantService
	Grants  *service.GrantService
	PATs    *service.PATService

	tmpl *template.Template
}

func New(auth *service.AuthService, users *service.UserService, tenants *service.TenantService, grants *service.GrantService, pats *service.PATService) (*Web, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Web{Auth: auth, Users: users, Tenants: tenants, Grants: grants, PATs: pats, tmpl: tmpl}, nil
}

// Router builds the web UI mux.
func (wb *Web) Router() *http.ServeMux {
	mux := http.NewServeMux()

	// staticFS's root is "static" (go:embed preserves the directory it's
	// declared in), so a request for /web/static/style.css must resolve to
	// "style.css" within an FS rooted at "static", not "static/style.css".
	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("web: static assets not embedded correctly: " + err.Error())
	}
	mux.Handle("GET /web/static/", http.StripPrefix("/web/static/", http.FileServerFS(staticRoot)))

	mux.HandleFunc("GET /web/login", wb.handleLoginPage)
	mux.HandleFunc("POST /web/login", wb.handleLoginSubmit)
	mux.HandleFunc("POST /web/logout", wb.handleLogout)

	mux.HandleFunc("GET /web/pats", wb.requireAuth(wb.handlePATsPage))
	mux.HandleFunc("POST /web/pats", wb.requireAuth(wb.handlePATsCreate))
	mux.HandleFunc("POST /web/pats/{id}/revoke", wb.requireAuth(wb.handlePATsRevoke))

	mux.HandleFunc("GET /web/tenants", wb.requireAdmin(wb.handleTenantsPage))
	mux.HandleFunc("POST /web/tenants", wb.requireAdmin(wb.handleTenantsCreate))
	mux.HandleFunc("POST /web/tenants/{tenantId}/delete", wb.requireAdmin(wb.handleTenantsDelete))

	mux.HandleFunc("GET /web/users", wb.requireAdmin(wb.handleUsersPage))
	mux.HandleFunc("POST /web/users", wb.requireAdmin(wb.handleUsersCreate))
	mux.HandleFunc("GET /web/users/{subject}", wb.requireAdmin(wb.handleUserDetailPage))
	mux.HandleFunc("POST /web/users/{subject}/delete", wb.requireAdmin(wb.handleUsersDelete))
	mux.HandleFunc("POST /web/users/{subject}/grants", wb.requireAdmin(wb.handleUsersGrant))
	mux.HandleFunc("POST /web/users/{subject}/grants/{tenantId}/revoke", wb.requireAdmin(wb.handleUsersRevokeGrant))
	mux.HandleFunc("POST /web/users/{subject}/pats", wb.requireAdmin(wb.handleUserPATsCreate))
	mux.HandleFunc("POST /web/users/{subject}/pats/{id}/revoke", wb.requireAdmin(wb.handleUserPATsRevoke))

	mux.HandleFunc("GET /web/", wb.handleRoot)

	return mux
}
