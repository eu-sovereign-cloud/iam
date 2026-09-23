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

func TestCreateTenant_TrimsBeforeConflictCheck(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateTenant{Tenants: memorystore.New(), Clock: fakeClock{now: time.Now()}, TenantRoles: memoryrbac.New()}

	_, err := create.Do(ctx, "tenant-1", "Tenant One")
	require.NoError(t, err)

	// Leading/trailing whitespace must not create a look-alike duplicate:
	// this is the exact bug report — "tenant-1" and "  tenant-1 " were
	// treated as different tenants because nothing trimmed the input.
	_, err = create.Do(ctx, "  tenant-1 ", "Tenant One again")
	require.ErrorIs(t, err, model.ErrConflict)
}

func TestCreateTenant_RequiresNonBlankID(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateTenant{Tenants: memorystore.New(), Clock: fakeClock{now: time.Now()}, TenantRoles: memoryrbac.New()}

	_, err := create.Do(ctx, "   ", "")
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestCreateTenant_RequiresDNS1123ID(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateTenant{Tenants: memorystore.New(), Clock: fakeClock{now: time.Now()}, TenantRoles: memoryrbac.New()}

	_, err := create.Do(ctx, "Tenant_One", "Tenant One")
	require.ErrorIs(t, err, model.ErrInvalid)

	_, err = create.Do(ctx, "tenant-one", "Tenant One")
	require.NoError(t, err, "a valid DNS-1123 label must still be accepted")
}

func TestCreateTenant_RequiresAdmin(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})
	create := &controller.CreateTenant{Tenants: memorystore.New(), Clock: fakeClock{now: time.Now()}, TenantRoles: memoryrbac.New()}

	_, err := create.Do(ctx, "tenant-1", "Tenant One")
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestCreateTenant_EnsuresTenantAdminRole(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	roles := memoryrbac.New()
	create := &controller.CreateTenant{Tenants: memorystore.New(), Clock: fakeClock{now: time.Now()}, TenantRoles: roles}

	_, err := create.Do(ctx, "tenant-1", "Tenant One")
	require.NoError(t, err)
	require.True(t, roles.HasTenantAdminRole("tenant-1"))
}

func TestDeleteTenant_RequiresNoGrants(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	roles := memoryrbac.New()
	create := &controller.CreateTenant{Tenants: store, Clock: clock, TenantRoles: roles}
	createGrant := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: roles}
	deleteTenant := &controller.DeleteTenant{Tenants: store, Grants: store, TenantRoles: roles}

	_, err := create.Do(ctx, "tenant-1", "Tenant One")
	require.NoError(t, err)
	require.NoError(t, store.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))
	_, err = createGrant.Do(ctx, "alice", "tenant-1", []string{"member"}, "admin", false)
	require.NoError(t, err)

	err = deleteTenant.Do(ctx, "tenant-1")
	require.ErrorIs(t, err, model.ErrConflict, "tenant with a remaining grant must not be deletable")

	require.NoError(t, store.DeleteGrant(ctx, "alice", "tenant-1"))
	require.NoError(t, deleteTenant.Do(ctx, "tenant-1"))
	require.False(t, roles.HasTenantAdminRole("tenant-1"))
}
