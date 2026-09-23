package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/adapter/memorycrypto"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/memoryrbac"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/memorystore"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/system"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/web"
)

// testWeb wires a full in-process Web against the in-memory adapters, the
// same in-memory-store approach internal/service's tests use.
type testWeb struct {
	wb         *web.Web
	mux        http.Handler
	adminPAT   string
	createUser *controller.CreateUser
	createPAT  *controller.CreatePAT
}

func newTestWeb(t *testing.T) *testWeb {
	t.Helper()
	ctx := context.Background()
	store := memorystore.New()

	clock := system.Clock{}
	createUser := &controller.CreateUser{Users: store, Clock: clock}
	getUser := &controller.GetUser{Users: store}
	listUsers := &controller.ListUsers{Users: store}
	deleteUser := &controller.DeleteUser{Users: store}

	roles := memoryrbac.New()

	createTenant := &controller.CreateTenant{Tenants: store, Clock: clock, TenantRoles: roles}
	getTenant := &controller.GetTenant{Tenants: store}
	listTenants := &controller.ListTenants{Tenants: store}
	deleteTenant := &controller.DeleteTenant{Tenants: store, Grants: store, TenantRoles: roles}
	repairTenant := &controller.RepairTenant{Tenants: store, Grants: store, TenantRoles: roles}

	createGrant := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: roles}
	listUserGrants := &controller.ListUserGrants{Grants: store}
	listTenantGrants := &controller.ListTenantGrants{Grants: store}
	deleteGrant := &controller.DeleteGrant{Grants: store, TenantRoles: roles}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: store, TenantRoles: roles}

	signer := memorycrypto.Signer{}
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

	wb, err := web.New(
		authenticateUser,
		createTenant, getTenant, listTenants, deleteTenant, repairTenant,
		createUser, getUser, listUsers, deleteUser,
		createGrant, listUserGrants, listTenantGrants, deleteGrant, setGrantAdmin,
		createPAT, listUserPATs, revokePAT,
	)
	require.NoError(t, err)

	return &testWeb{wb: wb, mux: wb.Router(), adminPAT: adminPAT, createUser: createUser, createPAT: createPAT}
}

func TestWebRoutesRenderWithoutError(t *testing.T) {
	tw := newTestWeb(t)
	mux := tw.mux

	// Unauthenticated: login page renders, protected pages redirect there.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/web/login", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Continue")

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/web/pats", nil))
	require.Equal(t, http.StatusSeeOther, rec.Code)

	// Authenticated (cookie-based) pages render.
	rec = doAuthed(mux, "/web/pats", tw.adminPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Your access tokens")

	rec = doAuthed(mux, "/web/tenants", tw.adminPAT)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doAuthed(mux, "/web/users", tw.adminPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "admin@example.com")
}

// TestWebTenantAdminManagesGrants mirrors internal/service/service_test.go's
// TestEndToEnd_CreateUserGrantIssuePATRevoke scenario, but through the web
// layer: a tenant admin (not a global admin) must be able to reach and use
// grant management for their own tenant, and nothing else.
func TestWebTenantAdminManagesGrants(t *testing.T) {
	tw := newTestWeb(t)
	mux := tw.mux
	ctx := context.Background()
	seedCtx := model.WithIdentity(ctx, model.User{Subject: "test-seed", Admin: true})

	// Admin creates a tenant and two non-admin users, then promotes bob to
	// tenant admin of tenant-1.
	rec := doAuthedForm(mux, "/web/tenants", tw.adminPAT, url.Values{"tenantId": {"tenant-1"}, "displayName": {"Tenant One"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)

	_, err := tw.createUser.Do(seedCtx, "bob@example.com", "Bob", false)
	require.NoError(t, err)
	_, bobPAT, err := tw.createPAT.Do(seedCtx, "bob@example.com", "bob-pat", nil, 0)
	require.NoError(t, err)
	_, err = tw.createUser.Do(seedCtx, "carol@example.com", "Carol", false)
	require.NoError(t, err)

	rec = doAuthedForm(mux, "/web/users/bob@example.com/grants", tw.adminPAT, url.Values{"tenantId": {"tenant-1"}, "roles": {"member"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	rec = doAuthedForm(mux, "/web/users/bob@example.com/grants/tenant-1/admin", tw.adminPAT, url.Values{"admin": {"true"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)

	// Bob, a tenant admin but not a global admin, can now reach the
	// Tenants pages at all - this is the core of the bug being fixed.
	rec = doAuthed(mux, "/web/pats", bobPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `href="/web/tenants"`, "the Tenants nav link must be visible to a tenant admin")
	require.NotContains(t, rec.Body.String(), `href="/web/users"`, "the Users nav link stays global-admin-only")

	rec = doAuthed(mux, "/web/tenants", bobPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "tenant-1")

	rec = doAuthed(mux, "/web/tenants/tenant-1", bobPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "bob@example.com")

	// Bob may grant carol access to tenant-1.
	rec = doAuthedForm(mux, "/web/tenants/tenant-1/grants", bobPAT, url.Values{"subject": {"carol@example.com"}, "roles": {"member"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)

	rec = doAuthed(mux, "/web/tenants/tenant-1", bobPAT)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "carol@example.com")

	// Bob may revoke carol's grant.
	rec = doAuthedForm(mux, "/web/tenants/tenant-1/grants/carol@example.com/revoke", bobPAT, nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)

	// Bob, a tenant admin but not a global admin, may also repair
	// tenant-1's RBAC state - RepairTenant.Do already used
	// requireTenantAdmin before this, it just had no web route at all.
	rec = doAuthedForm(mux, "/web/tenants/tenant-1/repair", bobPAT, nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)

	// Bob still holds no privilege to promote/demote tenant-admin status,
	// even for his own tenant (ADR 0016: global-admin-only). Like every
	// other mutation handler in this package, a rejected form submission
	// re-renders the originating page (200) with an inline error, rather
	// than a bare HTTP error status - checking for the "admin privileges
	// required" message is what actually proves the controller rejected
	// it, not just that the handler didn't crash.
	rec = doAuthedForm(mux, "/web/tenants/tenant-1/grants/carol@example.com/admin", bobPAT, url.Values{"admin": {"true"}})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "admin privileges required")

	// Bob may not create or delete tenants.
	rec = doAuthedForm(mux, "/web/tenants", bobPAT, url.Values{"tenantId": {"tenant-2"}})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "admin privileges required")
	rec = doAuthedForm(mux, "/web/tenants/tenant-1/delete", bobPAT, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "admin privileges required")

	// A subject with no grant at all on tenant-1 gets a real 403, not a
	// misleading 404, when trying to reach its detail page directly.
	_, err = tw.createUser.Do(seedCtx, "dave@example.com", "Dave", false)
	require.NoError(t, err)
	_, davePAT, err := tw.createPAT.Do(seedCtx, "dave@example.com", "dave-pat", nil, 0)
	require.NoError(t, err)
	rec = doAuthed(mux, "/web/tenants/tenant-1", davePAT)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func doAuthed(mux http.Handler, path, pat string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(&http.Cookie{Name: "iam_pat", Value: pat, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func doAuthedForm(mux http.Handler, path, pat string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "iam_pat", Value: pat, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
