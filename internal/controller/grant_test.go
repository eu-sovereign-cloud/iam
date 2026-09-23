package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/adapter/memoryrbac"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/memorystore"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestCreateGrant_RequiresExistingUserAndTenant(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: memoryrbac.New()}
	list := &controller.ListUserGrants{Grants: store}
	deleteGrant := &controller.DeleteGrant{Grants: store, TenantRoles: memoryrbac.New()}

	_, err := create.Do(ctx, "alice", "tenant-1", []string{"member"}, "admin", false)
	require.ErrorIs(t, err, model.ErrNotFound, "unknown user should be rejected")

	require.NoError(t, store.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))
	_, err = create.Do(ctx, "alice", "tenant-1", []string{"member"}, "admin", false)
	require.ErrorIs(t, err, model.ErrNotFound, "unknown tenant should be rejected")

	require.NoError(t, store.CreateTenant(ctx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	g, err := create.Do(ctx, "alice", "tenant-1", []string{"member", "viewer"}, "admin", false)
	require.NoError(t, err)
	require.Equal(t, "alice", g.Subject)
	require.Equal(t, "tenant-1", g.TenantID)
	require.Equal(t, []string{"member", "viewer"}, g.Roles)

	got, err := list.Do(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, got, 1)

	require.NoError(t, deleteGrant.Do(ctx, "alice", "tenant-1"))
	got, err = list.Do(ctx, "alice")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestCreateGrant_RequiresRole(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: memoryrbac.New()}

	require.NoError(t, store.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(ctx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))

	_, err := create.Do(ctx, "alice", "tenant-1", nil, "admin", false)
	require.ErrorIs(t, err, model.ErrInvalid)

	// Blank entries don't count either.
	_, err = create.Do(ctx, "alice", "tenant-1", []string{"  ", ""}, "admin", false)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestCreateGrant_SetsRoleAssignment(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	roles := memoryrbac.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: roles}

	require.NoError(t, store.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(ctx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	_, err := create.Do(ctx, "alice", "tenant-1", []string{"member", "viewer"}, "admin", false)
	require.NoError(t, err)

	got, err := roles.RoleAssignment("tenant-1", "alice")
	require.NoError(t, err)
	require.Equal(t, []string{"member", "viewer"}, got)
}

func TestListUserGrants_SelfOrAdmin(t *testing.T) {
	store := memorystore.New()
	list := &controller.ListUserGrants{Grants: store}

	selfCtx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})
	_, err := list.Do(selfCtx, "alice")
	require.NoError(t, err)

	otherCtx := model.WithIdentity(context.Background(), model.User{Subject: "bob"})
	_, err = list.Do(otherCtx, "alice")
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestSetGrantAdmin(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: memoryrbac.New()}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: store, TenantRoles: memoryrbac.New()}

	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	_, err := create.Do(adminCtx, "alice", "tenant-1", []string{"member"}, "admin", false)
	require.NoError(t, err)

	g, err := setGrantAdmin.Do(adminCtx, "alice", "tenant-1", true)
	require.NoError(t, err)
	require.True(t, g.Admin)

	g, err = setGrantAdmin.Do(adminCtx, "alice", "tenant-1", false)
	require.NoError(t, err)
	require.False(t, g.Admin)
}

func TestSetGrantAdmin_SwapsRoleAssignment(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	roles := memoryrbac.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: roles}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: store, TenantRoles: roles}

	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	_, err := create.Do(adminCtx, "alice", "tenant-1", []string{"member", "viewer"}, "admin", false)
	require.NoError(t, err)

	_, err = setGrantAdmin.Do(adminCtx, "alice", "tenant-1", true)
	require.NoError(t, err)
	got, err := roles.RoleAssignment("tenant-1", "alice")
	require.NoError(t, err)
	require.Equal(t, []string{model.TenantAdminRole}, got)

	_, err = setGrantAdmin.Do(adminCtx, "alice", "tenant-1", false)
	require.NoError(t, err)
	got, err = roles.RoleAssignment("tenant-1", "alice")
	require.NoError(t, err)
	require.Equal(t, []string{"member", "viewer"}, got, "demoting must revert the RoleAssignment to the grant's own roles")
}

func TestSetGrantAdmin_RequiresGlobalAdmin(t *testing.T) {
	store := memorystore.New()
	setGrantAdmin := &controller.SetGrantAdmin{Grants: store, TenantRoles: memoryrbac.New()}

	tenantAdminCtx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})
	_, err := setGrantAdmin.Do(tenantAdminCtx, "alice", "tenant-1", true)
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestListTenantGrants_TenantAdminOrGlobalAdmin(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	roles := memoryrbac.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: roles}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: store, TenantRoles: roles}
	list := &controller.ListTenantGrants{Grants: store}

	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "bob", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	_, err := create.Do(adminCtx, "alice", "tenant-1", []string{"member"}, "admin", false)
	require.NoError(t, err)
	_, err = setGrantAdmin.Do(adminCtx, "alice", "tenant-1", true)
	require.NoError(t, err)
	_, err = create.Do(adminCtx, "bob", "tenant-1", []string{"viewer"}, "admin", false)
	require.NoError(t, err)

	// A global admin sees every grant for the tenant.
	got, err := list.Do(adminCtx, "tenant-1")
	require.NoError(t, err)
	require.Len(t, got, 2)

	// Alice, tenant admin of tenant-1, may list its grants too.
	aliceCtx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})
	got, err = list.Do(aliceCtx, "tenant-1")
	require.NoError(t, err)
	require.Len(t, got, 2)

	// Bob holds a plain (non-admin) grant on tenant-1: not enough to list it.
	bobCtx := model.WithIdentity(context.Background(), model.User{Subject: "bob"})
	_, err = list.Do(bobCtx, "tenant-1")
	require.ErrorIs(t, err, model.ErrForbidden)

	// A stranger with no grant at all is rejected too.
	strangerCtx := model.WithIdentity(context.Background(), model.User{Subject: "carol"})
	_, err = list.Do(strangerCtx, "tenant-1")
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestCreateGrant_TenantAdminMayGrantOwnTenantOnly(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	roles := memoryrbac.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: roles}
	deleteGrant := &controller.DeleteGrant{Grants: store, TenantRoles: roles}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: store, TenantRoles: roles}

	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "carol", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-2", CreatedAt: clock.now}))
	_, err := create.Do(adminCtx, "alice", "tenant-1", []string{"member"}, "admin", false)
	require.NoError(t, err)
	_, err = setGrantAdmin.Do(adminCtx, "alice", "tenant-1", true)
	require.NoError(t, err)

	aliceCtx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})

	// Alice is tenant admin of tenant-1: she may grant/revoke access to it.
	_, err = create.Do(aliceCtx, "carol", "tenant-1", []string{"member"}, "alice", false)
	require.NoError(t, err)
	require.NoError(t, deleteGrant.Do(aliceCtx, "carol", "tenant-1"))

	// Alice holds no privilege over tenant-2.
	_, err = create.Do(aliceCtx, "carol", "tenant-2", []string{"member"}, "alice", false)
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestCreateGrant_AsAdminSkipsRoleRequirement(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	roles := memoryrbac.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: roles}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: store, TenantRoles: roles}

	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "bob", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))

	// A global admin may create a grant as tenant admin directly, with no
	// role at all - it would just be discarded the moment it's promoted
	// anyway (see TestSetGrantAdmin_SwapsRoleAssignment).
	g, err := create.Do(adminCtx, "alice", "tenant-1", nil, "admin", true)
	require.NoError(t, err)
	require.True(t, g.Admin)
	require.Empty(t, g.Roles)
	got, err := roles.RoleAssignment("tenant-1", "alice")
	require.NoError(t, err)
	require.Equal(t, []string{model.TenantAdminRole}, got)

	// A tenant admin (not global) may still not create a grant as tenant
	// admin, even for their own tenant - same rule SetGrantAdmin already
	// enforces (doc/adr/0016).
	_, err = setGrantAdmin.Do(adminCtx, "alice", "tenant-1", true) // already true; harmless
	require.NoError(t, err)
	aliceCtx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})
	_, err = create.Do(aliceCtx, "bob", "tenant-1", nil, "alice", true)
	require.ErrorIs(t, err, model.ErrForbidden)

	// A non-admin grant still requires a role.
	_, err = create.Do(adminCtx, "bob", "tenant-1", nil, "admin", false)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestCreateGrant_RequiresDNS1123Roles(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	create := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: memoryrbac.New()}

	require.NoError(t, store.CreateUser(adminCtx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, store.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))

	_, err := create.Do(adminCtx, "alice", "tenant-1", []string{"Member!"}, "admin", false)
	require.ErrorIs(t, err, model.ErrInvalid)

	_, err = create.Do(adminCtx, "alice", "tenant-1", []string{"member"}, "admin", false)
	require.NoError(t, err, "a valid DNS-1123 label role must still be accepted")
}
