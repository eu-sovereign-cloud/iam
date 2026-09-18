package web

import "net/http"

type tenantView struct {
	TenantID    string
	DisplayName string
	CreatedAt   string
}

func (wb *Web) handleTenantsPage(w http.ResponseWriter, r *http.Request) {
	wb.renderTenantsPage(w, r, "")
}

func (wb *Web) renderTenantsPage(w http.ResponseWriter, r *http.Request, errMsg string) {
	user := identityFromContext(r.Context())
	tenants, err := wb.Tenants.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views := make([]tenantView, 0, len(tenants))
	for _, t := range tenants {
		views = append(views, tenantView{TenantID: t.TenantID, DisplayName: t.DisplayName, CreatedAt: t.CreatedAt.Format(timeFormat)})
	}
	wb.render(w, "tenants.html", map[string]any{"User": user, "Tenants": views, "Error": errMsg})
}

func (wb *Web) handleTenantsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		wb.renderTenantsPage(w, r, "invalid form submission")
		return
	}
	if _, err := wb.Tenants.Create(r.Context(), r.FormValue("tenantId"), r.FormValue("displayName")); err != nil {
		wb.renderTenantsPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/tenants", http.StatusSeeOther)
}

func (wb *Web) handleTenantsDelete(w http.ResponseWriter, r *http.Request) {
	if err := wb.Tenants.Delete(r.Context(), r.PathValue("tenantId")); err != nil {
		wb.renderTenantsPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/tenants", http.StatusSeeOther)
}
