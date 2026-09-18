package web

import (
	"net/http"
	"time"
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
	user := identityFromContext(r.Context())
	pats, err := wb.PATs.ListBySubject(r.Context(), user.Subject)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views := make([]patView, 0, len(pats))
	for _, p := range pats {
		views = append(views, patView{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat), ExpiresAt: formatExpiry(p.ExpiresAt)})
	}
	wb.render(w, "pats.html", map[string]any{
		"User": user, "PATs": views, "NewSecret": newSecret, "Error": errMsg, "Active": "pats",
	})
}

func (wb *Web) handlePATsCreate(w http.ResponseWriter, r *http.Request) {
	user := identityFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		wb.renderPATsPage(w, r, "", "That submission did not come through. Try again.")
		return
	}
	var ttl time.Duration
	if raw := r.FormValue("ttl"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			wb.renderPATsPage(w, r, "", "Expiry must look like a duration, e.g. 720h.")
			return
		}
		ttl = parsed
	}
	_, raw, err := wb.PATs.Create(r.Context(), user.Subject, r.FormValue("name"), nil, ttl)
	if err != nil {
		wb.renderPATsPage(w, r, "", err.Error())
		return
	}
	wb.renderPATsPage(w, r, raw, "")
}

func (wb *Web) handlePATsRevoke(w http.ResponseWriter, r *http.Request) {
	user := identityFromContext(r.Context())
	id := r.PathValue("id")
	p, err := wb.PATs.Get(r.Context(), id)
	if err != nil || p.Subject != user.Subject {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := wb.PATs.Revoke(r.Context(), id); err != nil {
		wb.renderPATsPage(w, r, "", err.Error())
		return
	}
	http.Redirect(w, r, "/web/pats", http.StatusSeeOther)
}
