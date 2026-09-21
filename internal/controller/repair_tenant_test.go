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

func TestRepairTenant_ReappliesAndPrunes(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	roles := memoryrbac.New()
	createTenant := &controller.CreateTenant{Tenants: store, Clock: clock, TenantRoles: roles}
	createGrant := &controller.CreateGrant{Grants: store, Users: store, Tenants: store, Clock: clock, TenantRoles: roles}
	repair := &controller.RepairTenant{Tenants: store, Grants: store, TenantRoles: roles}

	_, err := createTenant.Do(ctx, "tenant-1", "Tenant One")
	require.NoError(t, err)
	require.NoError(t, store.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))
	_, err = createGrant.Do(ctx, "alice", "tenant-1", []string{"member"}, "admin")
	require.NoError(t, err)

	// Drift: the Role and RoleAssignment ecp holds diverge from IAM's
	// canonical state — a stale binding for a subject that no longer has
	// a Grant, and a Role that's gone missing.
	require.NoError(t, roles.DeleteTenantAdminRole(ctx, "tenant-1"))
	require.NoError(t, roles.SetRoleAssignment(ctx, "tenant-1", "alice", []string{"wrong-role"}))
	require.NoError(t, roles.SetRoleAssignment(ctx, "tenant-1", "orphan", []string{"member"}))

	require.NoError(t, repair.Do(ctx, "tenant-1"))

	require.True(t, roles.HasTenantAdminRole("tenant-1"))
	got, err := roles.RoleAssignment("tenant-1", "alice")
	require.NoError(t, err)
	require.Equal(t, []string{"member"}, got)
	_, err = roles.RoleAssignment("tenant-1", "orphan")
	require.Error(t, err, "an IAM-managed RoleAssignment with no matching Grant must be pruned")
}

func TestRepairTenant_RequiresTenantAdmin(t *testing.T) {
	store := memorystore.New()
	roles := memoryrbac.New()
	repair := &controller.RepairTenant{Tenants: store, Grants: store, TenantRoles: roles}

	otherCtx := model.WithIdentity(context.Background(), model.User{Subject: "bob"})
	err := repair.Do(otherCtx, "tenant-1")
	require.ErrorIs(t, err, model.ErrForbidden)
}
