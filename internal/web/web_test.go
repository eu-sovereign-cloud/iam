package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/eu-sovereign-cloud/iam/internal/adapter"
	"github.com/eu-sovereign-cloud/iam/internal/service"
	"github.com/eu-sovereign-cloud/iam/internal/web"
)

func TestWebRoutesRenderWithoutError(t *testing.T) {
	ctx := context.Background()
	client := k8sfake.NewClientset()
	store := adapter.NewStore(client, "iam-system")
	require.NoError(t, store.Load(ctx))

	clock := service.SystemClock{}
	userSvc := service.NewUserService(store, clock)
	tenantSvc := service.NewTenantService(store, clock)
	grantSvc := service.NewGrantService(store, store, store, clock)
	tokens := adapter.NewTokenGenerator()
	patSvc := service.NewPATService(store, tokens, clock)
	authSvc := service.NewAuthService(patSvc, store)

	admin, err := userSvc.Create(ctx, "admin@example.com", "Admin", true)
	require.NoError(t, err)
	_, adminPAT, err := patSvc.Create(ctx, admin.Subject, "bootstrap", nil, time.Hour)
	require.NoError(t, err)

	wb, err := web.New(authSvc, userSvc, tenantSvc, grantSvc, patSvc)
	require.NoError(t, err)
	mux := wb.Router()

	// Unauthenticated: login page renders, protected pages redirect there.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/web/login", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Log in")

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/web/pats", nil))
	require.Equal(t, http.StatusSeeOther, rec.Code)

	// Authenticated (cookie-based) pages render.
	rec = doAuthed(mux, http.MethodGet, "/web/pats", adminPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "My Personal Access Tokens")

	rec = doAuthed(mux, http.MethodGet, "/web/tenants", adminPAT)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doAuthed(mux, http.MethodGet, "/web/users", adminPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "admin@example.com")
}

func doAuthed(mux http.Handler, method, path, pat string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: "iam_pat", Value: pat, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
