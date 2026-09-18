package web

import "net/http"

type grantView struct {
	Subject  string
	TenantID string
}

type userView struct {
	Subject     string
	DisplayName string
	Admin       bool
	CreatedAt   string
	Grants      []grantView
}

func (wb *Web) handleUsersPage(w http.ResponseWriter, r *http.Request) {
	wb.renderUsersPage(w, r, "")
}

func (wb *Web) renderUsersPage(w http.ResponseWriter, r *http.Request, errMsg string) {
	caller := identityFromContext(r.Context())

	users, err := wb.Users.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tenants, err := wb.Tenants.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tenantViews := make([]tenantView, 0, len(tenants))
	for _, t := range tenants {
		tenantViews = append(tenantViews, tenantView{TenantID: t.TenantID, DisplayName: t.DisplayName})
	}

	userViews := make([]userView, 0, len(users))
	for _, u := range users {
		grants, err := wb.Grants.ListBySubject(r.Context(), u.Subject)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		grantViews := make([]grantView, 0, len(grants))
		for _, g := range grants {
			grantViews = append(grantViews, grantView{Subject: g.Subject, TenantID: g.TenantID})
		}
		userViews = append(userViews, userView{
			Subject: u.Subject, DisplayName: u.DisplayName, Admin: u.Admin,
			CreatedAt: u.CreatedAt.Format(timeFormat), Grants: grantViews,
		})
	}

	wb.render(w, "users.html", map[string]any{
		"User": caller, "Users": userViews, "Tenants": tenantViews, "Error": errMsg,
	})
}

func (wb *Web) handleUsersCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		wb.renderUsersPage(w, r, "invalid form submission")
		return
	}
	admin := r.FormValue("admin") == "true"
	if _, err := wb.Users.Create(r.Context(), r.FormValue("subject"), r.FormValue("displayName"), admin); err != nil {
		wb.renderUsersPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users", http.StatusSeeOther)
}

func (wb *Web) handleUsersDelete(w http.ResponseWriter, r *http.Request) {
	if err := wb.Users.Delete(r.Context(), r.PathValue("subject")); err != nil {
		wb.renderUsersPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users", http.StatusSeeOther)
}

func (wb *Web) handleUsersGrant(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		wb.renderUsersPage(w, r, "invalid form submission")
		return
	}
	caller := identityFromContext(r.Context())
	subject := r.PathValue("subject")
	if _, err := wb.Grants.Create(r.Context(), subject, r.FormValue("tenantId"), caller.Subject); err != nil {
		wb.renderUsersPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users", http.StatusSeeOther)
}

func (wb *Web) handleUsersRevokeGrant(w http.ResponseWriter, r *http.Request) {
	if err := wb.Grants.Delete(r.Context(), r.PathValue("subject"), r.PathValue("tenantId")); err != nil {
		wb.renderUsersPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users", http.StatusSeeOther)
}
