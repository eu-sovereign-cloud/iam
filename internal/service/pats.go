package service

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
	ExpiresAt string            `json:"expiresAt"`
}

type createPATResponse struct {
	patResponse
	Secret string `json:"secret"`
}

func (s *Service) handleCreatePAT(w http.ResponseWriter, r *http.Request) {
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

	p, raw, err := s.CreatePAT.Do(r.Context(), r.PathValue("subject"), req.Name, req.Scope, ttl)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, createPATResponse{patResponse: patToResponse(p), Secret: raw})
}

func (s *Service) handleListPATs(w http.ResponseWriter, r *http.Request) {
	pats, err := s.ListUserPATs.Do(r.Context(), r.PathValue("subject"))
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

func (s *Service) handleDeletePAT(w http.ResponseWriter, r *http.Request) {
	if err := s.RevokePAT.Do(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func patToResponse(p model.PAT) patResponse {
	return patResponse{
		ID:        p.ID,
		Subject:   p.Subject,
		Name:      p.Name,
		Scope:     p.Scope,
		CreatedAt: p.CreatedAt.Format(timeFormat),
		ExpiresAt: p.ExpiresAt.Format(timeFormat),
	}
}
