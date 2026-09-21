// Package service implements IAM's JSON REST API: routing, request
// decoding/response encoding, and bearer-auth middleware that resolves a
// PAT to its owning User. It is the HTTP/JSON bridge to internal/controller,
// which holds all the actual business logic, including authorization
// (admin-only vs self-or-admin), (ADR 0014).
package service

import (
	"net/http"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
)

// Service wires the REST handlers to the controller layer.
type Service struct {
	AuthenticateUser *controller.AuthenticateUser

	CreateTenant *controller.CreateTenant
	ListTenants  *controller.ListTenants
	DeleteTenant *controller.DeleteTenant

	CreateUser   *controller.CreateUser
	ListUsers    *controller.ListUsers
	SetUserAdmin *controller.SetUserAdmin
	DeleteUser   *controller.DeleteUser

	CreateGrant    *controller.CreateGrant
	ListUserGrants *controller.ListUserGrants
	DeleteGrant    *controller.DeleteGrant

	CreatePAT    *controller.CreatePAT
	ListUserPATs *controller.ListUserPATs
	RevokePAT    *controller.RevokePAT
}

// Router builds the full REST API mux (see the design plan's API surface
// section for the route list).
func (s *Service) Router() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/tenants", s.RequireAuth(s.handleCreateTenant))
	mux.HandleFunc("GET /api/v1/tenants", s.RequireAuth(s.handleListTenants))
	mux.HandleFunc("DELETE /api/v1/tenants/{tenantId}", s.RequireAuth(s.handleDeleteTenant))

	mux.HandleFunc("POST /api/v1/users", s.RequireAuth(s.handleCreateUser))
	mux.HandleFunc("GET /api/v1/users", s.RequireAuth(s.handleListUsers))
	mux.HandleFunc("PATCH /api/v1/users/{subject}", s.RequireAuth(s.handlePatchUser))
	mux.HandleFunc("DELETE /api/v1/users/{subject}", s.RequireAuth(s.handleDeleteUser))

	mux.HandleFunc("POST /api/v1/users/{subject}/grants", s.RequireAuth(s.handleCreateGrant))
	mux.HandleFunc("GET /api/v1/users/{subject}/grants", s.RequireAuth(s.handleListGrants))
	mux.HandleFunc("DELETE /api/v1/users/{subject}/grants/{tenantId}", s.RequireAuth(s.handleDeleteGrant))

	mux.HandleFunc("POST /api/v1/users/{subject}/pats", s.RequireAuth(s.handleCreatePAT))
	mux.HandleFunc("GET /api/v1/users/{subject}/pats", s.RequireAuth(s.handleListPATs))
	mux.HandleFunc("DELETE /api/v1/users/{subject}/pats/{id}", s.RequireAuth(s.handleDeletePAT))

	return mux
}
