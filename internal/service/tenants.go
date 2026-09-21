package service

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

func (s *Service) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req tenantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := s.CreateTenant.Do(r.Context(), req.TenantID, req.DisplayName)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tenantResponse{
		TenantID: t.TenantID, DisplayName: t.DisplayName, CreatedAt: t.CreatedAt.Format(timeFormat),
	})
}

func (s *Service) handleListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := s.ListTenants.Do(r.Context())
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

func (s *Service) handleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	if err := s.DeleteTenant.Do(r.Context(), r.PathValue("tenantId")); err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleRepairTenant(w http.ResponseWriter, r *http.Request) {
	if err := s.RepairTenant.Do(r.Context(), r.PathValue("tenantId")); err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
