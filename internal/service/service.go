// Package service implements IAM's JSON REST API: routing, request
// decoding/response encoding, and bearer-auth middleware that resolves a
// PAT to its owning User and enforces admin-only vs self-service access.
// It is the HTTP/JSON bridge to internal/controller, which holds all the
// actual business logic (ADR 0014).
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
	GetPAT       *controller.GetPAT
	RevokePAT    *controller.RevokePAT
}

// Router builds the full REST API mux (see the design plan's API surface
// section for the route list).
func (s *Service) Router() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/tenants", s.RequireAdmin(s.handleCreateTenant))
	mux.HandleFunc("GET /api/v1/tenants", s.RequireAdmin(s.handleListTenants))
	mux.HandleFunc("DELETE /api/v1/tenants/{tenantId}", s.RequireAdmin(s.handleDeleteTenant))

	mux.HandleFunc("POST /api/v1/users", s.RequireAdmin(s.handleCreateUser))
	mux.HandleFunc("GET /api/v1/users", s.RequireAdmin(s.handleListUsers))
	mux.HandleFunc("PATCH /api/v1/users/{subject}", s.RequireAdmin(s.handlePatchUser))
	mux.HandleFunc("DELETE /api/v1/users/{subject}", s.RequireAdmin(s.handleDeleteUser))

	mux.HandleFunc("POST /api/v1/users/{subject}/grants", s.RequireAdmin(s.handleCreateGrant))
	mux.HandleFunc("GET /api/v1/users/{subject}/grants", s.RequireSelfOrAdmin(s.handleListGrants))
	mux.HandleFunc("DELETE /api/v1/users/{subject}/grants/{tenantId}", s.RequireAdmin(s.handleDeleteGrant))

	mux.HandleFunc("POST /api/v1/users/{subject}/pats", s.RequireSelfOrAdmin(s.handleCreatePAT))
	mux.HandleFunc("GET /api/v1/users/{subject}/pats", s.RequireSelfOrAdmin(s.handleListPATs))
	mux.HandleFunc("DELETE /api/v1/users/{subject}/pats/{id}", s.RequireSelfOrAdmin(s.handleDeletePAT))

	return mux
}
