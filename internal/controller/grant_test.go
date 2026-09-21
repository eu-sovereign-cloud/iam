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
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	tenants := newFakeTenantStore()
	grants := newFakeGrantStore()
	create := controller.NewCreateGrant(grants, users, tenants, clock)
	list := controller.NewListUserGrants(grants)
	deleteGrant := controller.NewDeleteGrant(grants)

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
