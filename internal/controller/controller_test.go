package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	tokenSvc  *service.TokenService
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
	tokens := adapter.NewTokenGenerator()

	clock := service.SystemClock{}
	userSvc := service.NewUserService(store, clock)
	tenantSvc := service.NewTenantService(store, clock)
	grantSvc := service.NewGrantService(store, store, store, clock)
	patSvc := service.NewPATService(store, tokens, clock)
	authSvc := service.NewAuthService(patSvc, store)
	tokenSvc := service.NewTokenService(patSvc, store, store, signer, clock, "https://iam.example.com", "ecp-gateway", 15*time.Minute)

	admin, err := userSvc.Create(ctx, "admin@example.com", "Admin", true)
	require.NoError(t, err)
	_, adminPAT, err := patSvc.Create(ctx, admin.Subject, "bootstrap", nil, 0)
	require.NoError(t, err)

	return &testStack{
		ctl: &controller.Controller{
			Auth: authSvc, Users: userSvc, Tenants: tenantSvc, Grants: grantSvc, PATs: patSvc, Tokens: tokenSvc,
		},
		adminPAT: adminPAT, tokenSvc: tokenSvc, patSvc: patSvc, userSvc: userSvc, tenantSvc: tenantSvc, grantSvc: grantSvc,
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

func TestEndToEnd_CreateUserGrantExchangeRevoke(t *testing.T) {
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

	// Bob (self-service) creates his own PAT.
	rec = doJSON(t, mux, http.MethodPost, "/api/v1/users/bob@example.com/pats", bobPAT, map[string]any{"name": "laptop"})
	require.Equal(t, http.StatusCreated, rec.Code)
	var created struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&created))
	require.NotEmpty(t, created.Secret)

	// Exchange the new PAT for a JWT carrying the granted tenant.
	rec = doJSON(t, mux, http.MethodPost, "/api/v1/tokens", "", map[string]string{"pat": created.Secret})
	require.Equal(t, http.StatusOK, rec.Code)
	var issued struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&issued))
	require.Equal(t, "Bearer", issued.TokenType)
	require.NotEmpty(t, issued.AccessToken)

	claims, err := parseUnverified(issued.AccessToken)
	require.NoError(t, err)
	require.Equal(t, "bob@example.com", claims.Subject)
	require.Equal(t, []string{"tenant-1"}, claims.Tenants)

	// Bob cannot manage another subject's PATs.
	rec = doJSON(t, mux, http.MethodGet, "/api/v1/users/admin@example.com/pats", bobPAT, nil)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// Revoke bob's PAT; the exchange must now fail.
	rec = doJSON(t, mux, http.MethodDelete, "/api/v1/users/bob@example.com/pats/"+created.ID, bobPAT, nil)
	require.Equal(t, http.StatusNoContent, rec.Code)

	rec = doJSON(t, mux, http.MethodPost, "/api/v1/tokens", "", map[string]string{"pat": created.Secret})
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func parseUnverified(token string) (*model.Claims, error) {
	claims := &model.Claims{}
	_, _, err := jwtParser.ParseUnverified(token, claims)
	return claims, err
}
