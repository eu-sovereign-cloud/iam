package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/eu-sovereign-cloud/iam/internal/adapter/kubecrypt"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/kubestore"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/system"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/service"
)

var jwtParser = jwt.NewParser()

// testStack wires a full, in-process instance of the service against a
// fake Kubernetes clientset (no real cluster needed) so these tests
// exercise real routing, real auth middleware and real business logic —
// only the Kubernetes API itself is faked.
type testStack struct {
	svc        *service.Service
	adminPAT   string
	createUser *controller.CreateUser
	createPAT  *controller.CreatePAT
}

func newTestStack(t *testing.T) *testStack {
	t.Helper()
	ctx := context.Background()
	client := k8sfake.NewClientset()

	store := kubestore.New(client, "iam-system")
	require.NoError(t, store.Load(ctx))

	signer, err := kubecrypt.LoadOrCreate(ctx, client, "iam-system")
	require.NoError(t, err)

	clock := system.Clock{}

	createTenant := &controller.CreateTenant{Tenants: store, Clock: clock}
	listTenants := &controller.ListTenants{Tenants: store}
	deleteTenant := &controller.DeleteTenant{Tenants: store}

	createUser := &controller.CreateUser{Users: store, Clock: clock}
	listUsers := &controller.ListUsers{Users: store}
	setUserAdmin := &controller.SetUserAdmin{Users: store}
	deleteUser := &controller.DeleteUser{Users: store}

	createGrant := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock}
	listUserGrants := &controller.ListUserGrants{Grants: store}
	deleteGrant := &controller.DeleteGrant{Grants: store}

	createPAT := &controller.CreatePAT{PATs: store, Grants: store, Signer: signer, Clock: clock, Issuer: "https://iam.example.com", Audience: []string{"ecp-gateway"}}
	listUserPATs := &controller.ListUserPATs{PATs: store}
	revokePAT := &controller.RevokePAT{PATs: store}

	authenticatePAT := &controller.AuthenticatePAT{PATs: store, Signer: signer, Clock: clock}
	authenticateUser := &controller.AuthenticateUser{PATs: authenticatePAT, Users: store}

	// Seeding the first admin bypasses normal auth (there's no admin yet to
	// authenticate as), the same way controller.EnsureBootstrapAdmin does.
	seedCtx := model.WithIdentity(ctx, model.User{Subject: "test-seed", Admin: true})
	admin, err := createUser.Do(seedCtx, "admin@example.com", "Admin", true)
	require.NoError(t, err)
	_, adminPAT, err := createPAT.Do(seedCtx, admin.Subject, "bootstrap", nil, 0)
	require.NoError(t, err)

	return &testStack{
		svc: &service.Service{
			AuthenticateUser: authenticateUser,
			CreateTenant:     createTenant, ListTenants: listTenants, DeleteTenant: deleteTenant,
			CreateUser: createUser, ListUsers: listUsers, SetUserAdmin: setUserAdmin, DeleteUser: deleteUser,
			CreateGrant: createGrant, ListUserGrants: listUserGrants, DeleteGrant: deleteGrant,
			CreatePAT: createPAT, ListUserPATs: listUserPATs, RevokePAT: revokePAT,
		},
		adminPAT:   adminPAT,
		createUser: createUser,
		createPAT:  createPAT,
	}
}

func doJSON(t *testing.T, mux http.Handler, method, path, bearer string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestEndToEnd_CreateUserGrantIssuePATRevoke(t *testing.T) {
	stack := newTestStack(t)
	mux := stack.svc.Router()

	// Non-admin cannot create tenants.
	seedCtx := model.WithIdentity(context.Background(), model.User{Subject: "test-seed", Admin: true})
	nonAdmin, err := stack.createUser.Do(seedCtx, "bob@example.com", "Bob", false)
	require.NoError(t, err)
	_, bobPAT, err := stack.createPAT.Do(seedCtx, nonAdmin.Subject, "bob-pat", nil, 0)
	require.NoError(t, err)

	rec := doJSON(t, mux, http.MethodPost, "/api/v1/tenants", bobPAT, map[string]string{"tenantId": "tenant-1"})
	require.Equal(t, http.StatusForbidden, rec.Code)

	// Admin creates a tenant.
	rec = doJSON(t, mux, http.MethodPost, "/api/v1/tenants", stack.adminPAT, map[string]string{"tenantId": "tenant-1", "displayName": "Tenant One"})
	require.Equal(t, http.StatusCreated, rec.Code)

	// Admin grants bob access to it.
	rec = doJSON(t, mux, http.MethodPost, "/api/v1/users/bob@example.com/grants", stack.adminPAT, map[string]string{"tenantId": "tenant-1"})
	require.Equal(t, http.StatusCreated, rec.Code)

	// Bob (self-service) issues himself a new PAT. Per ADR 0012 the PAT
	// itself is the signed JWT — there is no separate exchange step.
	rec = doJSON(t, mux, http.MethodPost, "/api/v1/users/bob@example.com/pats", bobPAT, map[string]any{"name": "laptop"})
	require.Equal(t, http.StatusCreated, rec.Code)
	var created struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&created))
	require.NotEmpty(t, created.Secret)

	claims, err := parseUnverified(created.Secret)
	require.NoError(t, err)
	require.Equal(t, "bob@example.com", claims.Subject)
	require.Equal(t, []string{"tenant-1"}, claims.Tenants)

	// The newly issued PAT authenticates directly against IAM's own API.
	rec = doJSON(t, mux, http.MethodGet, "/api/v1/users/bob@example.com/pats", created.Secret, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// Bob cannot manage another subject's PATs.
	rec = doJSON(t, mux, http.MethodGet, "/api/v1/users/admin@example.com/pats", bobPAT, nil)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// Revoke the new PAT; the same signed JWT must now be rejected by IAM
	// itself, even though its signature and expiry are still technically
	// valid (ADR 0012's revocation gap is scoped to *other* verifiers).
	rec = doJSON(t, mux, http.MethodDelete, "/api/v1/users/bob@example.com/pats/"+created.ID, bobPAT, nil)
	require.Equal(t, http.StatusNoContent, rec.Code)

	rec = doJSON(t, mux, http.MethodGet, "/api/v1/users/bob@example.com/pats", created.Secret, nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func parseUnverified(token string) (*model.Claims, error) {
	claims := &model.Claims{}
	_, _, err := jwtParser.ParseUnverified(token, claims)
	return claims, err
}
