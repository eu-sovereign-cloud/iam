package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type tenantView struct {
	TenantID    string
	DisplayName string
	CreatedAt   string
}

// tenantAdminOf returns the tenant IDs the caller (from ctx) holds an
// admin Grant for (ADR 0016). Meaningful for non-global-admins; a global
// admin doesn't need it to reach any tenant, but calling it for one is
// harmless.
func (wb *Web) tenantAdminOf(ctx context.Context) ([]string, error) {
	caller := model.IdentityFromContext(ctx)
	grants, err := wb.ListUserGrants.Do(ctx, caller.Subject) // self, always allowed
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, g := range grants {
		if g.Admin {
			ids = append(ids, g.TenantID)
		}
	}
	return ids, nil
}

func (wb *Web) handleTenantsPage(w http.ResponseWriter, r *http.Request) {
	wb.renderTenantsPage(w, r, "")
}

func (wb *Web) renderTenantsPage(w http.ResponseWriter, r *http.Request, errMsg string) {
	user := model.IdentityFromContext(r.Context())

	// Global admins see every tenant; everyone else only sees the
	// tenant(s) they administer (there's no page for a plain member to
	// browse tenants they hold no admin privilege over).
	var views []tenantView
	var adminOf []string
	if user.Admin {
		tenants, err := wb.ListTenants.Do(r.Context())
		if err != nil {
			http.Error(w, err.Error(), statusFor(err))
			return
		}
		views = make([]tenantView, 0, len(tenants))
		for _, t := range tenants {
			views = append(views, tenantView{TenantID: t.TenantID, DisplayName: t.DisplayName, CreatedAt: t.CreatedAt.Format(timeFormat)})
		}
	} else {
		var err error
		adminOf, err = wb.tenantAdminOf(r.Context())
		if err != nil {
			http.Error(w, err.Error(), statusFor(err))
			return
		}
		views = make([]tenantView, 0, len(adminOf))
		for _, tenantID := range adminOf {
			t, err := wb.GetTenant.Do(r.Context(), tenantID)
			if err != nil {
				http.Error(w, err.Error(), statusFor(err))
				return
			}
			views = append(views, tenantView{TenantID: t.TenantID, DisplayName: t.DisplayName, CreatedAt: t.CreatedAt.Format(timeFormat)})
		}
	}

	wb.render(w, "tenants.html", map[string]any{
		"User": user, "Tenants": views, "Error": errMsg, "Active": "tenants",
		"IsTenantAdmin": user.Admin || len(adminOf) > 0,
	})
}

func (wb *Web) handleTenantsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		wb.renderTenantsPage(w, r, "That submission did not come through. Try again.")
		return
	}
	if _, err := wb.CreateTenant.Do(r.Context(), r.FormValue("tenantId"), r.FormValue("displayName")); err != nil {
		wb.renderTenantsPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/tenants", http.StatusSeeOther)
}

func (wb *Web) handleTenantsDelete(w http.ResponseWriter, r *http.Request) {
	if err := wb.DeleteTenant.Do(r.Context(), r.PathValue("tenantId")); err != nil {
		wb.renderTenantsPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/tenants", http.StatusSeeOther)
}

func (wb *Web) handleTenantDetailPage(w http.ResponseWriter, r *http.Request) {
	wb.renderTenantDetailPage(w, r, r.PathValue("tenantId"), "")
}

func (wb *Web) renderTenantDetailPage(w http.ResponseWriter, r *http.Request, tenantID, errMsg string) {
	caller := model.IdentityFromContext(r.Context())

	// ListTenantGrants is what actually authorizes this page (global
	// admin or tenant admin of tenantID, ADR 0016) - do it first so an
	// unauthorized caller gets a real 403 via statusFor, not a page built
	// from a partial/forbidden GetTenant call.
	grants, err := wb.ListTenantGrants.Do(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}
	grantViews := make([]grantView, 0, len(grants))
	for _, g := range grants {
		grantViews = append(grantViews, grantView{Subject: g.Subject, TenantID: g.TenantID, Admin: g.Admin, Roles: g.Roles})
	}

	tenant, err := wb.GetTenant.Do(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}

	adminOf, err := wb.tenantAdminOf(r.Context())
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}

	wb.render(w, "tenant_detail.html", map[string]any{
		"User": caller,
		"Tenant": tenantView{
			TenantID: tenant.TenantID, DisplayName: tenant.DisplayName, CreatedAt: tenant.CreatedAt.Format(timeFormat),
		},
		"Grants":        grantViews,
		"Error":         errMsg,
		"Active":        "tenants",
		"IsTenantAdmin": caller.Admin || len(adminOf) > 0,
	})
}

func (wb *Web) handleTenantsGrant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantId")
	if err := r.ParseForm(); err != nil {
		wb.renderTenantDetailPage(w, r, tenantID, "That submission did not come through. Try again.")
		return
	}
	caller := model.IdentityFromContext(r.Context())
	roles := strings.Split(r.FormValue("roles"), ",")
	admin := r.FormValue("admin") == "true"
	if _, err := wb.CreateGrant.Do(r.Context(), r.FormValue("subject"), tenantID, roles, caller.Subject, admin); err != nil {
		wb.renderTenantDetailPage(w, r, tenantID, err.Error())
		return
	}
	http.Redirect(w, r, "/web/tenants/"+tenantID, http.StatusSeeOther)
}

func (wb *Web) handleTenantsRevokeGrant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantId")
	if err := wb.DeleteGrant.Do(r.Context(), r.PathValue("subject"), tenantID); err != nil {
		wb.renderTenantDetailPage(w, r, tenantID, err.Error())
		return
	}
	http.Redirect(w, r, "/web/tenants/"+tenantID, http.StatusSeeOther)
}

func (wb *Web) handleTenantsRepair(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantId")
	if err := wb.RepairTenant.Do(r.Context(), tenantID); err != nil {
		wb.renderTenantDetailPage(w, r, tenantID, err.Error())
		return
	}
	http.Redirect(w, r, "/web/tenants/"+tenantID, http.StatusSeeOther)
}

func (wb *Web) handleTenantsSetGrantAdmin(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantId")
	if err := r.ParseForm(); err != nil {
		wb.renderTenantDetailPage(w, r, tenantID, "That submission did not come through. Try again.")
		return
	}
	admin := r.FormValue("admin") == "true"
	if _, err := wb.SetGrantAdmin.Do(r.Context(), r.PathValue("subject"), tenantID, admin); err != nil {
		wb.renderTenantDetailPage(w, r, tenantID, err.Error())
		return
	}
	http.Redirect(w, r, "/web/tenants/"+tenantID, http.StatusSeeOther)
}
