package controller

import "net/http"

type tenantRequest struct {
	TenantID    string `json:"tenantId"`
	DisplayName string `json:"displayName"`
}

type tenantResponse struct {
	TenantID    string `json:"tenantId"`
	DisplayName string `json:"displayName"`
	CreatedAt   string `json:"createdAt"`
}

func (c *Controller) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req tenantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := c.Tenants.Create(r.Context(), req.TenantID, req.DisplayName)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tenantResponse{
		TenantID: t.TenantID, DisplayName: t.DisplayName, CreatedAt: t.CreatedAt.Format(timeFormat),
	})
}

func (c *Controller) handleListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := c.Tenants.List(r.Context())
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	out := make([]tenantResponse, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, tenantResponse{TenantID: t.TenantID, DisplayName: t.DisplayName, CreatedAt: t.CreatedAt.Format(timeFormat)})
	}
	writeJSON(w, http.StatusOK, out)
}

func (c *Controller) handleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	if err := c.Tenants.Delete(r.Context(), r.PathValue("tenantId")); err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
