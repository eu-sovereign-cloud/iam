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
	userViews := make([]userView, 0, len(users))
	for _, u := range users {
		userViews = append(userViews, userView{
			Subject: u.Subject, DisplayName: u.DisplayName, Admin: u.Admin,
			CreatedAt: u.CreatedAt.Format(timeFormat),
		})
	}

	wb.render(w, "users.html", map[string]any{
		"User": caller, "Users": userViews, "Error": errMsg, "Active": "users",
	})
}

func (wb *Web) handleUsersCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		wb.renderUsersPage(w, r, "That submission did not come through. Try again.")
		return
	}
	admin := r.FormValue("admin") == "true"
	if _, err := wb.Users.Create(r.Context(), r.FormValue("subject"), r.FormValue("displayName"), admin); err != nil {
		wb.renderUsersPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users", http.StatusSeeOther)
}

func (wb *Web) handleUserDetailPage(w http.ResponseWriter, r *http.Request) {
	wb.renderUserDetailPage(w, r, r.PathValue("subject"), "")
}

func (wb *Web) renderUserDetailPage(w http.ResponseWriter, r *http.Request, subject, errMsg string) {
	caller := identityFromContext(r.Context())

	user, err := wb.Users.Get(r.Context(), subject)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	grants, err := wb.Grants.ListBySubject(r.Context(), subject)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	grantViews := make([]grantView, 0, len(grants))
	for _, g := range grants {
		grantViews = append(grantViews, grantView{Subject: g.Subject, TenantID: g.TenantID})
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

	wb.render(w, "user_detail.html", map[string]any{
		"User": caller,
		"Subject": userView{
			Subject: user.Subject, DisplayName: user.DisplayName, Admin: user.Admin,
			CreatedAt: user.CreatedAt.Format(timeFormat),
		},
		"Grants":  grantViews,
		"Tenants": tenantViews,
		"Error":   errMsg,
		"Active":  "users",
	})
}

func (wb *Web) handleUsersDelete(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := wb.Users.Delete(r.Context(), subject); err != nil {
		wb.renderUserDetailPage(w, r, subject, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users", http.StatusSeeOther)
}

func (wb *Web) handleUsersGrant(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := r.ParseForm(); err != nil {
		wb.renderUserDetailPage(w, r, subject, "That submission did not come through. Try again.")
		return
	}
	caller := identityFromContext(r.Context())
	if _, err := wb.Grants.Create(r.Context(), subject, r.FormValue("tenantId"), caller.Subject); err != nil {
		wb.renderUserDetailPage(w, r, subject, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users/"+subject, http.StatusSeeOther)
}

func (wb *Web) handleUsersRevokeGrant(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := wb.Grants.Delete(r.Context(), subject, r.PathValue("tenantId")); err != nil {
		wb.renderUserDetailPage(w, r, subject, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users/"+subject, http.StatusSeeOther)
}
