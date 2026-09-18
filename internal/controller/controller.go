// Package controller implements IAM's JSON REST API: routing, request
// decoding/response encoding, and bearer-auth middleware that resolves a
// PAT to its owning User and enforces admin-only vs self-service access.
package controller

import (
	"net/http"

	"github.com/eu-sovereign-cloud/iam/internal/service"
)

// Controller wires the REST handlers to the service layer.
type Controller struct {
	Auth    *service.AuthService
	Users   *service.UserService
	Tenants *service.TenantService
	Grants  *service.GrantService
	PATs    *service.PATService
}

// Router builds the full REST API mux (see the design plan's API surface
// section for the route list).
func (c *Controller) Router() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/tenants", c.RequireAdmin(c.handleCreateTenant))
	mux.HandleFunc("GET /api/v1/tenants", c.RequireAdmin(c.handleListTenants))
	mux.HandleFunc("DELETE /api/v1/tenants/{tenantId}", c.RequireAdmin(c.handleDeleteTenant))

	mux.HandleFunc("POST /api/v1/users", c.RequireAdmin(c.handleCreateUser))
	mux.HandleFunc("GET /api/v1/users", c.RequireAdmin(c.handleListUsers))
	mux.HandleFunc("PATCH /api/v1/users/{subject}", c.RequireAdmin(c.handlePatchUser))
	mux.HandleFunc("DELETE /api/v1/users/{subject}", c.RequireAdmin(c.handleDeleteUser))

	mux.HandleFunc("POST /api/v1/users/{subject}/grants", c.RequireAdmin(c.handleCreateGrant))
	mux.HandleFunc("GET /api/v1/users/{subject}/grants", c.RequireSelfOrAdmin(c.handleListGrants))
	mux.HandleFunc("DELETE /api/v1/users/{subject}/grants/{tenantId}", c.RequireAdmin(c.handleDeleteGrant))

	mux.HandleFunc("POST /api/v1/users/{subject}/pats", c.RequireSelfOrAdmin(c.handleCreatePAT))
	mux.HandleFunc("GET /api/v1/users/{subject}/pats", c.RequireSelfOrAdmin(c.handleListPATs))
	mux.HandleFunc("DELETE /api/v1/users/{subject}/pats/{id}", c.RequireSelfOrAdmin(c.handleDeletePAT))

	return mux
}
