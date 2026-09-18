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

func (wb *Web) renderPATsPage(w http.ResponseWriter, r *http.Request, newSecret, errMsg string) {
	user := identityFromContext(r.Context())
	pats, err := wb.PATs.ListBySubject(r.Context(), user.Subject)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views := make([]patView, 0, len(pats))
	for _, p := range pats {
		v := patView{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt.Format(timeFormat)}
		if p.ExpiresAt != nil {
			v.ExpiresAt = p.ExpiresAt.Format(timeFormat)
		}
		views = append(views, v)
	}
	wb.render(w, "pats.html", map[string]any{
		"User": user, "PATs": views, "NewSecret": newSecret, "Error": errMsg,
	})
}

func (wb *Web) handlePATsCreate(w http.ResponseWriter, r *http.Request) {
	user := identityFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		wb.renderPATsPage(w, r, "", "invalid form submission")
		return
	}
	var ttl time.Duration
	if raw := r.FormValue("ttl"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			wb.renderPATsPage(w, r, "", "ttl must be a duration string, e.g. \"720h\"")
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
