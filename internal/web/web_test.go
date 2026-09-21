package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/eu-sovereign-cloud/iam/internal/adapter"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/web"
)

func TestWebRoutesRenderWithoutError(t *testing.T) {
	ctx := context.Background()
	client := k8sfake.NewClientset()
	store := adapter.NewStore(client, "iam-system")
	require.NoError(t, store.Load(ctx))

	clock := adapter.SystemClock{}
	createUser := &controller.CreateUser{Users: store, Clock: clock}
	getUser := &controller.GetUser{Users: store}
	listUsers := &controller.ListUsers{Users: store}
	deleteUser := &controller.DeleteUser{Users: store}

	createTenant := &controller.CreateTenant{Tenants: store, Clock: clock}
	listTenants := &controller.ListTenants{Tenants: store}
	deleteTenant := &controller.DeleteTenant{Tenants: store}

	createGrant := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock}
	listUserGrants := &controller.ListUserGrants{Grants: store}
	deleteGrant := &controller.DeleteGrant{Grants: store}

	signer, err := adapter.LoadOrCreateSigner(ctx, client, "iam-system")
	require.NoError(t, err)
	createPAT := &controller.CreatePAT{PATs: store, Grants: store, Signer: signer, Clock: clock, Issuer: "https://iam.example.com", Audience: []string{"ecp-gateway"}}
	listUserPATs := &controller.ListUserPATs{PATs: store}
	getPAT := &controller.GetPAT{PATs: store}
	revokePAT := &controller.RevokePAT{PATs: store}

	authenticatePAT := &controller.AuthenticatePAT{PATs: store, Signer: signer, Clock: clock}
	authenticateUser := &controller.AuthenticateUser{PATs: authenticatePAT, Users: store}

	admin, err := createUser.Do(ctx, "admin@example.com", "Admin", true)
	require.NoError(t, err)
	_, adminPAT, err := createPAT.Do(ctx, admin.Subject, "bootstrap", nil, 0)
	require.NoError(t, err)

	wb, err := web.New(
		authenticateUser,
		createTenant, listTenants, deleteTenant,
		createUser, getUser, listUsers, deleteUser,
		createGrant, listUserGrants, deleteGrant,
		createPAT, listUserPATs, getPAT, revokePAT,
	)
	require.NoError(t, err)
	mux := wb.Router()

	// Unauthenticated: login page renders, protected pages redirect there.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/web/login", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Continue")

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/web/pats", nil))
	require.Equal(t, http.StatusSeeOther, rec.Code)

	// Authenticated (cookie-based) pages render.
	rec = doAuthed(mux, http.MethodGet, "/web/pats", adminPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Your access tokens")

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
