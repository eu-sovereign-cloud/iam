package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestCreateGrant_RequiresExistingUserAndTenant(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	tenants := newFakeTenantStore()
	grants := newFakeGrantStore()
	create := &controller.CreateGrant{Grants: grants, Users: users, Tenants: tenants, Clock: clock}
	list := &controller.ListUserGrants{Grants: grants}
	deleteGrant := &controller.DeleteGrant{Grants: grants}

	_, err := create.Do(ctx, "alice", "tenant-1", "admin")
	require.ErrorIs(t, err, model.ErrNotFound, "unknown user should be rejected")

	require.NoError(t, users.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))
	_, err = create.Do(ctx, "alice", "tenant-1", "admin")
	require.ErrorIs(t, err, model.ErrNotFound, "unknown tenant should be rejected")

	require.NoError(t, tenants.CreateTenant(ctx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	g, err := create.Do(ctx, "alice", "tenant-1", "admin")
	require.NoError(t, err)
	require.Equal(t, "alice", g.Subject)
	require.Equal(t, "tenant-1", g.TenantID)

	got, err := list.Do(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, got, 1)

	require.NoError(t, deleteGrant.Do(ctx, "alice", "tenant-1"))
	got, err = list.Do(ctx, "alice")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestListUserGrants_SelfOrAdmin(t *testing.T) {
	grants := newFakeGrantStore()
	list := &controller.ListUserGrants{Grants: grants}

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
	users := newFakeUserStore()
	tenants := newFakeTenantStore()
	grants := newFakeGrantStore()
	create := &controller.CreateGrant{Grants: grants, Users: users, Tenants: tenants, Clock: clock}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: grants}

	require.NoError(t, users.CreateUser(adminCtx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, tenants.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	_, err := create.Do(adminCtx, "alice", "tenant-1", "admin")
	require.NoError(t, err)

	g, err := setGrantAdmin.Do(adminCtx, "alice", "tenant-1", true)
	require.NoError(t, err)
	require.True(t, g.Admin)

	g, err = setGrantAdmin.Do(adminCtx, "alice", "tenant-1", false)
	require.NoError(t, err)
	require.False(t, g.Admin)
}

func TestSetGrantAdmin_RequiresGlobalAdmin(t *testing.T) {
	grants := newFakeGrantStore()
	setGrantAdmin := &controller.SetGrantAdmin{Grants: grants}

	tenantAdminCtx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})
	_, err := setGrantAdmin.Do(tenantAdminCtx, "alice", "tenant-1", true)
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestCreateGrant_TenantAdminMayGrantOwnTenantOnly(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	tenants := newFakeTenantStore()
	grants := newFakeGrantStore()
	create := &controller.CreateGrant{Grants: grants, Users: users, Tenants: tenants, Clock: clock}
	deleteGrant := &controller.DeleteGrant{Grants: grants}
	setGrantAdmin := &controller.SetGrantAdmin{Grants: grants}

	require.NoError(t, users.CreateUser(adminCtx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, users.CreateUser(adminCtx, model.User{Subject: "carol", CreatedAt: clock.now}))
	require.NoError(t, tenants.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	require.NoError(t, tenants.CreateTenant(adminCtx, model.Tenant{TenantID: "tenant-2", CreatedAt: clock.now}))
	_, err := create.Do(adminCtx, "alice", "tenant-1", "admin")
	require.NoError(t, err)
	_, err = setGrantAdmin.Do(adminCtx, "alice", "tenant-1", true)
	require.NoError(t, err)

	aliceCtx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})

	// Alice is tenant admin of tenant-1: she may grant/revoke access to it.
	_, err = create.Do(aliceCtx, "carol", "tenant-1", "alice")
	require.NoError(t, err)
	require.NoError(t, deleteGrant.Do(aliceCtx, "carol", "tenant-1"))

	// Alice holds no privilege over tenant-2.
	_, err = create.Do(aliceCtx, "carol", "tenant-2", "alice")
	require.ErrorIs(t, err, model.ErrForbidden)
}
