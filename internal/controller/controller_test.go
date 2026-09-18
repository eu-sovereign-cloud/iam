package controller_test

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

	"github.com/eu-sovereign-cloud/iam/internal/adapter"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/service"
)

var jwtParser = jwt.NewParser()

// testStack wires a full, in-process instance of the service against a
// fake Kubernetes clientset (no real cluster needed) so controller tests
// exercise real routing, real auth middleware and real business logic —
// only the Kubernetes API itself is faked.
type testStack struct {
	ctl       *controller.Controller
	adminPAT  string
	patSvc    *service.PATService
	userSvc   *service.UserService
	tenantSvc *service.TenantService
	grantSvc  *service.GrantService
}

func newTestStack(t *testing.T) *testStack {
	t.Helper()
	ctx := context.Background()
	client := k8sfake.NewClientset()

	store := adapter.NewStore(client, "iam-system")
	require.NoError(t, store.Load(ctx))

	signer, err := adapter.LoadOrCreateSigner(ctx, client, "iam-system")
	require.NoError(t, err)

	clock := service.SystemClock{}
	userSvc := service.NewUserService(store, clock)
	tenantSvc := service.NewTenantService(store, clock)
	grantSvc := service.NewGrantService(store, store, store, clock)
	patSvc := service.NewPATService(store, store, signer, clock, "https://iam.example.com", "ecp-gateway")
	authSvc := service.NewAuthService(patSvc, store)

	admin, err := userSvc.Create(ctx, "admin@example.com", "Admin", true)
	require.NoError(t, err)
	_, adminPAT, err := patSvc.Create(ctx, admin.Subject, "bootstrap", nil, 0)
	require.NoError(t, err)

	return &testStack{
		ctl: &controller.Controller{
			Auth: authSvc, Users: userSvc, Tenants: tenantSvc, Grants: grantSvc, PATs: patSvc,
		},
		adminPAT: adminPAT, patSvc: patSvc, userSvc: userSvc, tenantSvc: tenantSvc, grantSvc: grantSvc,
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
	mux := stack.ctl.Router()

	// Non-admin cannot create tenants.
	nonAdmin, err := stack.userSvc.Create(context.Background(), "bob@example.com", "Bob", false)
	require.NoError(t, err)
	_, bobPAT, err := stack.patSvc.Create(context.Background(), nonAdmin.Subject, "bob-pat", nil, 0)
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
