package controller

import (
	"net/http"
	"time"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type createPATRequest struct {
	Name  string            `json:"name"`
	Scope *model.TokenScope `json:"scope"`
	TTL   string            `json:"ttl"` // e.g. "720h"; empty means the PAT never expires
}

type patResponse struct {
	ID        string            `json:"id"`
	Subject   string            `json:"subject"`
	Name      string            `json:"name"`
	Scope     *model.TokenScope `json:"scope,omitempty"`
	CreatedAt string            `json:"createdAt"`
	ExpiresAt *string           `json:"expiresAt,omitempty"`
}

type createPATResponse struct {
	patResponse
	Secret string `json:"secret"`
}

func (c *Controller) handleCreatePAT(w http.ResponseWriter, r *http.Request) {
	var req createPATRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var ttl time.Duration
	if req.TTL != "" {
		parsed, err := time.ParseDuration(req.TTL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "ttl must be a duration string, e.g. \"720h\"")
			return
		}
		ttl = parsed
	}

	p, raw, err := c.PATs.Create(r.Context(), r.PathValue("subject"), req.Name, req.Scope, ttl)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, createPATResponse{patResponse: patToResponse(p), Secret: raw})
}

func (c *Controller) handleListPATs(w http.ResponseWriter, r *http.Request) {
	pats, err := c.PATs.ListBySubject(r.Context(), r.PathValue("subject"))
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	out := make([]patResponse, 0, len(pats))
	for _, p := range pats {
		out = append(out, patToResponse(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (c *Controller) handleDeletePAT(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := c.PATs.Get(r.Context(), id)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	if p.Subject != r.PathValue("subject") {
		writeError(w, http.StatusNotFound, model.ErrNotFound.Error())
		return
	}
	if err := c.PATs.Revoke(r.Context(), id); err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func patToResponse(p model.PAT) patResponse {
	resp := patResponse{
		ID:        p.ID,
		Subject:   p.Subject,
		Name:      p.Name,
		Scope:     p.Scope,
		CreatedAt: p.CreatedAt.Format(timeFormat),
	}
	if p.ExpiresAt != nil {
		formatted := p.ExpiresAt.Format(timeFormat)
		resp.ExpiresAt = &formatted
	}
	return resp
}
