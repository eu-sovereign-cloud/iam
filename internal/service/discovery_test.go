package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func discoveryMux(t *testing.T, stack *testStack) http.Handler {
	t.Helper()
	mux := stack.svc.Router()
	stack.svc.RegisterDiscoveryRoutes(mux)
	return mux
}

func TestOpenIDConfiguration(t *testing.T) {
	stack := newTestStack(t)
	mux := discoveryMux(t, stack)

	rec := doJSON(t, mux, http.MethodGet, "/.well-known/openid-configuration", "", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var doc struct {
		Issuer                           string   `json:"issuer"`
		JWKSURI                          string   `json:"jwks_uri"`
		UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
		IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&doc))
	require.Equal(t, "https://iam.example.com", doc.Issuer)
	require.Equal(t, "https://iam.example.com/.well-known/jwks.json", doc.JWKSURI)
	require.Equal(t, "https://iam.example.com/userinfo", doc.UserinfoEndpoint)
	require.Equal(t, []string{"ES256"}, doc.IDTokenSigningAlgValuesSupported)
}

func TestJWKS(t *testing.T) {
	stack := newTestStack(t)
	mux := discoveryMux(t, stack)

	rec := doJSON(t, mux, http.MethodGet, "/.well-known/jwks.json", "", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "max-age=300", rec.Header().Get("Cache-Control"))

	var set struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
		} `json:"keys"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&set))
	require.Len(t, set.Keys, 1)
	require.Equal(t, "EC", set.Keys[0].Kty)
	require.NotEmpty(t, set.Keys[0].Kid)
}

func TestUserinfo(t *testing.T) {
	stack := newTestStack(t)
	mux := discoveryMux(t, stack)

	seedCtx := model.WithIdentity(context.Background(), model.User{Subject: "test-seed", Admin: true})
	_, err := stack.createUser.Do(seedCtx, "alice@example.com", "Alice", false)
	require.NoError(t, err)
	pat, raw, err := stack.createPAT.Do(seedCtx, "alice@example.com", "laptop", nil, 0)
	require.NoError(t, err)

	// Valid, live PAT.
	rec := doJSON(t, mux, http.MethodGet, "/userinfo", raw, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var info struct {
		Subject string `json:"sub"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&info))
	require.Equal(t, "alice@example.com", info.Subject)

	// Missing bearer token.
	rec = doJSON(t, mux, http.MethodGet, "/userinfo", "", nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// Garbage bearer token.
	rec = doJSON(t, mux, http.MethodGet, "/userinfo", "not-a-real-token", nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, `Bearer error="invalid_token"`, rec.Header().Get("WWW-Authenticate"))

	// Revoked PAT: the JWT itself is still validly signed and unexpired,
	// but /userinfo must reflect the revocation immediately (issue #2).
	require.NoError(t, stack.revokePAT.Do(seedCtx, pat.ID))
	rec = doJSON(t, mux, http.MethodGet, "/userinfo", raw, nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
