package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/service"
)

func TestGrantServiceCreate_RequiresExistingUserAndTenant(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	tenants := newFakeTenantStore()
	svc := service.NewGrantService(newFakeGrantStore(), users, tenants, clock)

	_, err := svc.Create(ctx, "alice", "tenant-1", "admin")
	require.ErrorIs(t, err, model.ErrNotFound, "unknown user should be rejected")

	require.NoError(t, users.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))
	_, err = svc.Create(ctx, "alice", "tenant-1", "admin")
	require.ErrorIs(t, err, model.ErrNotFound, "unknown tenant should be rejected")

	require.NoError(t, tenants.CreateTenant(ctx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	g, err := svc.Create(ctx, "alice", "tenant-1", "admin")
	require.NoError(t, err)
	require.Equal(t, "alice", g.Subject)
	require.Equal(t, "tenant-1", g.TenantID)

	list, err := svc.ListBySubject(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, svc.Delete(ctx, "alice", "tenant-1"))
	list, err = svc.ListBySubject(ctx, "alice")
	require.NoError(t, err)
	require.Empty(t, list)
}
