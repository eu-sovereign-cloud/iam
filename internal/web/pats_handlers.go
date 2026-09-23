package web

import (
	"net/http"
	"time"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type patView struct {
	ID        string
	Name      string
	CreatedAt string
	ExpiresAt string
}

func (wb *Web) handlePATsPage(w http.ResponseWriter, r *http.Request) {
	wb.renderPATsPage(w, r, "", "")
}

// noExpiryDisplayThreshold: PATs created with no requested TTL get a
// 100-year expiry so the JWT's mandatory exp claim still holds (ADR 0012).
// Anything further out than this is shown as "Never" rather than a literal
// date nobody will ever see arrive.
const noExpiryDisplayThreshold = 50 * 365 * 24 * time.Hour

func formatExpiry(t time.Time) string {
	if time.Until(t) > noExpiryDisplayThreshold {
		return ""
	}
	return t.Format(timeFormat)
}

func (wb *Web) renderPATsPage(w http.ResponseWriter, r *http.Request, newSecret, errMsg string) {
	user := model.IdentityFromContext(r.Context())
	pats, err := wb.ListUserPATs.Do(r.Context(), user.Subject)
	if err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}
	views := make([]patView, 0, len(pats))
	for _, p := range pats {
		views = append(views, patView{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat), ExpiresAt: formatExpiry(p.ExpiresAt)})
	}

	var isTenantAdmin bool
	if user.Admin {
		isTenantAdmin = true
	} else {
		adminOf, err := wb.tenantAdminOf(r.Context())
		if err != nil {
			http.Error(w, err.Error(), statusFor(err))
			return
		}
		isTenantAdmin = len(adminOf) > 0
	}

	wb.render(w, "pats.html", map[string]any{
		"User": user, "PATs": views, "NewSecret": newSecret, "Error": errMsg, "Active": "pats",
		"IsTenantAdmin": isTenantAdmin,
	})
}

// parseTTL reads the "ttl" form field shared by the self-service and
// admin-on-behalf-of issue-token forms.
func parseTTL(r *http.Request) (time.Duration, error) {
	raw := r.FormValue("ttl")
	if raw == "" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}

func (wb *Web) handlePATsCreate(w http.ResponseWriter, r *http.Request) {
	user := model.IdentityFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		wb.renderPATsPage(w, r, "", "That submission did not come through. Try again.")
		return
	}
	ttl, err := parseTTL(r)
	if err != nil {
		wb.renderPATsPage(w, r, "", "Expiry must look like a duration, e.g. 720h.")
		return
	}
	_, raw, err := wb.CreatePAT.Do(r.Context(), user.Subject, r.FormValue("name"), nil, ttl)
	if err != nil {
		wb.renderPATsPage(w, r, "", err.Error())
		return
	}
	wb.renderPATsPage(w, r, raw, "")
}

func (wb *Web) handlePATsRevoke(w http.ResponseWriter, r *http.Request) {
	if err := wb.RevokePAT.Do(r.Context(), r.PathValue("id")); err != nil {
		wb.renderPATsPage(w, r, "", err.Error())
		return
	}
	http.Redirect(w, r, "/web/pats", http.StatusSeeOther)
}
