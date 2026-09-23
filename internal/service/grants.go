package service

import (
	"net/http"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type createGrantRequest struct {
	TenantID string   `json:"tenantId"`
	Roles    []string `json:"roles"`
	Admin    bool     `json:"admin"`
}

type patchGrantRequest struct {
	Admin *bool `json:"admin"`
}

type grantResponse struct {
	Subject   string   `json:"subject"`
	TenantID  string   `json:"tenantId"`
	GrantedAt string   `json:"grantedAt"`
	GrantedBy string   `json:"grantedBy"`
	Admin     bool     `json:"admin"`
	Roles     []string `json:"roles"`
}

func grantToResponse(g model.Grant) grantResponse {
	return grantResponse{
		Subject: g.Subject, TenantID: g.TenantID,
		GrantedAt: g.GrantedAt.Format(timeFormat), GrantedBy: g.GrantedBy,
		Admin: g.Admin, Roles: g.Roles,
	}
}

func (s *Service) handleCreateGrant(w http.ResponseWriter, r *http.Request) {
	var req createGrantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	grantedBy := model.IdentityFromContext(r.Context()).Subject
	g, err := s.CreateGrant.Do(r.Context(), r.PathValue("subject"), req.TenantID, req.Roles, grantedBy, req.Admin)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, grantToResponse(g))
}

func (s *Service) handleListGrants(w http.ResponseWriter, r *http.Request) {
	grants, err := s.ListUserGrants.Do(r.Context(), r.PathValue("subject"))
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	out := make([]grantResponse, 0, len(grants))
	for _, g := range grants {
		out = append(out, grantToResponse(g))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Service) handlePatchGrant(w http.ResponseWriter, r *http.Request) {
	var req patchGrantRequest
	if err := decodeJSON(r, &req); err != nil || req.Admin == nil {
		writeError(w, http.StatusBadRequest, "body must set \"admin\"")
		return
	}
	g, err := s.SetGrantAdmin.Do(r.Context(), r.PathValue("subject"), r.PathValue("tenantId"), *req.Admin)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, grantToResponse(g))
}

func (s *Service) handleDeleteGrant(w http.ResponseWriter, r *http.Request) {
	if err := s.DeleteGrant.Do(r.Context(), r.PathValue("subject"), r.PathValue("tenantId")); err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
