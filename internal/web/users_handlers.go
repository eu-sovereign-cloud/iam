package web

import (
	"net/http"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type grantView struct {
	Subject  string
	TenantID string
	Admin    bool
	Roles    []string
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
	caller := model.IdentityFromContext(r.Context())

	users, err := wb.ListUsers.Do(r.Context())
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
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
	if _, err := wb.CreateUser.Do(r.Context(), r.FormValue("subject"), r.FormValue("displayName"), admin); err != nil {
		wb.renderUsersPage(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/web/users", http.StatusSeeOther)
}

func (wb *Web) handleUserDetailPage(w http.ResponseWriter, r *http.Request) {
	wb.renderUserDetailPage(w, r, r.PathValue("subject"), "", "")
}

func (wb *Web) renderUserDetailPage(w http.ResponseWriter, r *http.Request, subject, newSecret, errMsg string) {
	caller := model.IdentityFromContext(r.Context())

	user, err := wb.GetUser.Do(r.Context(), subject)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	grants, err := wb.ListUserGrants.Do(r.Context(), subject)
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}
	grantViews := make([]grantView, 0, len(grants))
	for _, g := range grants {
		grantViews = append(grantViews, grantView{Subject: g.Subject, TenantID: g.TenantID, Admin: g.Admin, Roles: g.Roles})
	}

	tenants, err := wb.ListTenants.Do(r.Context())
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}
	tenantViews := make([]tenantView, 0, len(tenants))
	for _, t := range tenants {
		tenantViews = append(tenantViews, tenantView{TenantID: t.TenantID, DisplayName: t.DisplayName})
	}

	pats, err := wb.ListUserPATs.Do(r.Context(), subject)
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}
	patViews := make([]patView, 0, len(pats))
	for _, p := range pats {
		patViews = append(patViews, patView{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat), ExpiresAt: formatExpiry(p.ExpiresAt)})
	}

	wb.render(w, "user_detail.html", map[string]any{
		"User": caller,
		"Subject": userView{
			Subject: user.Subject, DisplayName: user.DisplayName, Admin: user.Admin,
			CreatedAt: user.CreatedAt.Format(timeFormat),
		},
		"Grants":    grantViews,
		"Tenants":   tenantViews,
		"PATs":      patViews,
		"NewSecret": newSecret,
		"Error":     errMsg,
		"Active":    "users",
	})
}

func (wb *Web) handleUserPATsCreate(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := r.ParseForm(); err != nil {
		wb.renderUserDetailPage(w, r, subject, "", "That submission did not come through. Try again.")
		return
	}
	ttl, err := parseTTL(r)
	if err != nil {
		wb.renderUserDetailPage(w, r, subject, "", "Expiry must look like a duration, e.g. 720h.")
		return
	}
	_, raw, err := wb.CreatePAT.Do(r.Context(), subject, r.FormValue("name"), nil, ttl)
	if err != nil {
		wb.renderUserDetailPage(w, r, subject, "", err.Error())
		return
	}
	wb.renderUserDetailPage(w, r, subject, raw, "")
}

func (wb *Web) handleUserPATsRevoke(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := wb.RevokePAT.Do(r.Context(), r.PathValue("id")); err != nil {
		wb.renderUserDetailPage(w, r, subject, "", err.Error())
		return
	}
	http.Redirect(w, r, "/web/users/"+subject, http.StatusSeeOther)
}

func (wb *Web) handleUsersDelete(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := wb.DeleteUser.Do(r.Context(), subject); err != nil {
		wb.renderUserDetailPage(w, r, subject, "", err.Error())
		return
	}
	http.Redirect(w, r, "/web/users", http.StatusSeeOther)
}

func (wb *Web) handleUsersGrant(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := r.ParseForm(); err != nil {
		wb.renderUserDetailPage(w, r, subject, "", "That submission did not come through. Try again.")
		return
	}
	caller := model.IdentityFromContext(r.Context())
	roles := strings.Split(r.FormValue("roles"), ",")
	if _, err := wb.CreateGrant.Do(r.Context(), subject, r.FormValue("tenantId"), roles, caller.Subject); err != nil {
		wb.renderUserDetailPage(w, r, subject, "", err.Error())
		return
	}
	http.Redirect(w, r, "/web/users/"+subject, http.StatusSeeOther)
}

func (wb *Web) handleUsersRevokeGrant(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := wb.DeleteGrant.Do(r.Context(), subject, r.PathValue("tenantId")); err != nil {
		wb.renderUserDetailPage(w, r, subject, "", err.Error())
		return
	}
	http.Redirect(w, r, "/web/users/"+subject, http.StatusSeeOther)
}

func (wb *Web) handleUsersSetGrantAdmin(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	if err := r.ParseForm(); err != nil {
		wb.renderUserDetailPage(w, r, subject, "", "That submission did not come through. Try again.")
		return
	}
	admin := r.FormValue("admin") == "true"
	if _, err := wb.SetGrantAdmin.Do(r.Context(), subject, r.PathValue("tenantId"), admin); err != nil {
		wb.renderUserDetailPage(w, r, subject, "", err.Error())
		return
	}
	http.Redirect(w, r, "/web/users/"+subject, http.StatusSeeOther)
}
