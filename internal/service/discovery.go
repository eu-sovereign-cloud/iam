package service

import "net/http"

// discoveryDocument is the body of GET /.well-known/openid-configuration —
// a practical subset of OIDC discovery metadata (RFC/OIDC Discovery 1.0):
// just enough for a verifier to find IAM's JWKS and userinfo endpoints and
// confirm the signing algorithm, not full OIDC (IAM has no /authorize or
// /token endpoint — PATs are minted directly, ADR 0012).
type discoveryDocument struct {
	Issuer                           string   `json:"issuer"`
	JWKSURI                          string   `json:"jwks_uri"`
	UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

type userinfoResponse struct {
	Subject string `json:"sub"`
}

// RegisterDiscoveryRoutes adds IAM's public, unauthenticated-by-RequireAuth
// discovery endpoints (issue #2, ADR 0019) directly onto mux — they live at
// fixed top-level paths (OIDC convention), not under /api/, so they're
// registered on the shared root mux rather than inside Router().
func (s *Service) RegisterDiscoveryRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/openid-configuration", s.handleOpenIDConfiguration)
	mux.HandleFunc("GET /.well-known/jwks.json", s.handleJWKS)
	mux.HandleFunc("GET /userinfo", s.handleUserinfo)
}

func (s *Service) handleOpenIDConfiguration(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, discoveryDocument{
		Issuer:                           s.Issuer,
		JWKSURI:                          s.Issuer + "/.well-known/jwks.json",
		UserinfoEndpoint:                 s.Issuer + "/userinfo",
		ResponseTypesSupported:           []string{"id_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"ES256"},
	})
}

func (s *Service) handleJWKS(w http.ResponseWriter, r *http.Request) {
	set, err := s.GetJWKS.Do(r.Context())
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	// The key rotates rarely (ADR 0005); a short cache is a cheap way to
	// spare verifiers a round trip on every request.
	w.Header().Set("Cache-Control", "max-age=300")
	writeJSON(w, http.StatusOK, set)
}

// handleUserinfo lets a verifier that has already checked a JWT's
// signature offline (via JWKS) confirm it's still live — the one thing
// offline verification can't see is a PAT revoked after issuance (issue
// #2). It authenticates via the bearer token under test itself, not
// RequireAuth/AuthenticateUser.
func (s *Service) handleUserinfo(w http.ResponseWriter, r *http.Request) {
	raw := bearerToken(r)
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	pat, err := s.AuthenticatePAT.Do(r.Context(), raw)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeError(w, http.StatusUnauthorized, "invalid or revoked token")
		return
	}
	writeJSON(w, http.StatusOK, userinfoResponse{Subject: pat.Subject})
}
