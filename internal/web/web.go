// Package web implements IAM's minimal server-rendered admin/self-service
// UI: plain HTML templates, one small stylesheet, no client-side framework
// (see ADR 0002 and the design plan's low-tech UI constraint).
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

const cookieName = "iam_pat"

// Web wires the HTML handlers to the controller layer. Authentication is
// cookie-based: the cookie value *is* the bearer PAT (ADR: no server-side
// session store, consistent with the k8s-only storage constraint).
type Web struct {
	AuthenticateUser *controller.AuthenticateUser

	CreateTenant *controller.CreateTenant
	GetTenant    *controller.GetTenant
	ListTenants  *controller.ListTenants
	DeleteTenant *controller.DeleteTenant
	RepairTenant *controller.RepairTenant

	CreateUser *controller.CreateUser
	GetUser    *controller.GetUser
	ListUsers  *controller.ListUsers
	DeleteUser *controller.DeleteUser

	CreateGrant      *controller.CreateGrant
	ListUserGrants   *controller.ListUserGrants
	ListTenantGrants *controller.ListTenantGrants
	DeleteGrant      *controller.DeleteGrant
	SetGrantAdmin    *controller.SetGrantAdmin

	CreatePAT    *controller.CreatePAT
	ListUserPATs *controller.ListUserPATs
	RevokePAT    *controller.RevokePAT

	tmpl *template.Template
}

func New(
	authenticateUser *controller.AuthenticateUser,
	createTenant *controller.CreateTenant, getTenant *controller.GetTenant, listTenants *controller.ListTenants, deleteTenant *controller.DeleteTenant, repairTenant *controller.RepairTenant,
	createUser *controller.CreateUser, getUser *controller.GetUser, listUsers *controller.ListUsers, deleteUser *controller.DeleteUser,
	createGrant *controller.CreateGrant, listUserGrants *controller.ListUserGrants, listTenantGrants *controller.ListTenantGrants, deleteGrant *controller.DeleteGrant, setGrantAdmin *controller.SetGrantAdmin,
	createPAT *controller.CreatePAT, listUserPATs *controller.ListUserPATs, revokePAT *controller.RevokePAT,
) (*Web, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Web{
		AuthenticateUser: authenticateUser,
		CreateTenant:     createTenant, GetTenant: getTenant, ListTenants: listTenants, DeleteTenant: deleteTenant, RepairTenant: repairTenant,
		CreateUser: createUser, GetUser: getUser, ListUsers: listUsers, DeleteUser: deleteUser,
		CreateGrant: createGrant, ListUserGrants: listUserGrants, ListTenantGrants: listTenantGrants, DeleteGrant: deleteGrant, SetGrantAdmin: setGrantAdmin,
		CreatePAT: createPAT, ListUserPATs: listUserPATs, RevokePAT: revokePAT,
		tmpl: tmpl,
	}, nil
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

	mux.HandleFunc("GET /web/tenants", wb.requireAuth(wb.handleTenantsPage))
	mux.HandleFunc("POST /web/tenants", wb.requireAuth(wb.handleTenantsCreate))
	mux.HandleFunc("GET /web/tenants/{tenantId}", wb.requireAuth(wb.handleTenantDetailPage))
	mux.HandleFunc("POST /web/tenants/{tenantId}/delete", wb.requireAuth(wb.handleTenantsDelete))
	mux.HandleFunc("POST /web/tenants/{tenantId}/grants", wb.requireAuth(wb.handleTenantsGrant))
	mux.HandleFunc("POST /web/tenants/{tenantId}/grants/{subject}/revoke", wb.requireAuth(wb.handleTenantsRevokeGrant))
	mux.HandleFunc("POST /web/tenants/{tenantId}/grants/{subject}/admin", wb.requireAuth(wb.handleTenantsSetGrantAdmin))
	mux.HandleFunc("POST /web/tenants/{tenantId}/repair", wb.requireAuth(wb.handleTenantsRepair))

	mux.HandleFunc("GET /web/users", wb.requireAuth(wb.handleUsersPage))
	mux.HandleFunc("POST /web/users", wb.requireAuth(wb.handleUsersCreate))
	mux.HandleFunc("GET /web/users/{subject}", wb.requireAuth(wb.handleUserDetailPage))
	mux.HandleFunc("POST /web/users/{subject}/delete", wb.requireAuth(wb.handleUsersDelete))
	mux.HandleFunc("POST /web/users/{subject}/grants", wb.requireAuth(wb.handleUsersGrant))
	mux.HandleFunc("POST /web/users/{subject}/grants/{tenantId}/revoke", wb.requireAuth(wb.handleUsersRevokeGrant))
	mux.HandleFunc("POST /web/users/{subject}/grants/{tenantId}/admin", wb.requireAuth(wb.handleUsersSetGrantAdmin))
	mux.HandleFunc("POST /web/users/{subject}/pats", wb.requireAuth(wb.handleUserPATsCreate))
	mux.HandleFunc("POST /web/users/{subject}/pats/{id}/revoke", wb.requireAuth(wb.handleUserPATsRevoke))

	mux.HandleFunc("GET /web/", wb.handleRoot)

	return mux
}
