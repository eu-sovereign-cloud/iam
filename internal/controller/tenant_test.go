package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestCreateTenant_TrimsBeforeConflictCheck(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateTenant{Tenants: newFakeTenantStore(), Clock: fakeClock{now: time.Now()}}

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
	create := &controller.CreateTenant{Tenants: newFakeTenantStore(), Clock: fakeClock{now: time.Now()}}

	_, err := create.Do(ctx, "   ", "")
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestCreateTenant_RequiresAdmin(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})
	create := &controller.CreateTenant{Tenants: newFakeTenantStore(), Clock: fakeClock{now: time.Now()}}

	_, err := create.Do(ctx, "tenant-1", "Tenant One")
	require.ErrorIs(t, err, model.ErrForbidden)
}
