package controller

import "net/http"

type createGrantRequest struct {
	TenantID string `json:"tenantId"`
}

type grantResponse struct {
	Subject   string `json:"subject"`
	TenantID  string `json:"tenantId"`
	GrantedAt string `json:"grantedAt"`
	GrantedBy string `json:"grantedBy"`
}

func (c *Controller) handleCreateGrant(w http.ResponseWriter, r *http.Request) {
	var req createGrantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	grantedBy := identityFromContext(r.Context()).Subject
	g, err := c.Grants.Create(r.Context(), r.PathValue("subject"), req.TenantID, grantedBy)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, grantResponse{
		Subject: g.Subject, TenantID: g.TenantID, GrantedAt: g.GrantedAt.Format(timeFormat), GrantedBy: g.GrantedBy,
	})
}

func (c *Controller) handleListGrants(w http.ResponseWriter, r *http.Request) {
	grants, err := c.Grants.ListBySubject(r.Context(), r.PathValue("subject"))
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	out := make([]grantResponse, 0, len(grants))
	for _, g := range grants {
		out = append(out, grantResponse{Subject: g.Subject, TenantID: g.TenantID, GrantedAt: g.GrantedAt.Format(timeFormat), GrantedBy: g.GrantedBy})
	}
	writeJSON(w, http.StatusOK, out)
}

func (c *Controller) handleDeleteGrant(w http.ResponseWriter, r *http.Request) {
	if err := c.Grants.Delete(r.Context(), r.PathValue("subject"), r.PathValue("tenantId")); err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
