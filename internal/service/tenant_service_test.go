package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/service"
)

func TestTenantServiceCreate_TrimsBeforeConflictCheck(t *testing.T) {
	ctx := context.Background()
	svc := service.NewTenantService(newFakeTenantStore(), fakeClock{now: time.Now()})

	_, err := svc.Create(ctx, "tenant-1", "Tenant One")
	require.NoError(t, err)

	// Leading/trailing whitespace must not create a look-alike duplicate:
	// this is the exact bug report — "tenant-1" and "  tenant-1 " were
	// treated as different tenants because nothing trimmed the input.
	_, err = svc.Create(ctx, "  tenant-1 ", "Tenant One again")
	require.ErrorIs(t, err, model.ErrConflict)
}

func TestTenantServiceCreate_RequiresNonBlankID(t *testing.T) {
	ctx := context.Background()
	svc := service.NewTenantService(newFakeTenantStore(), fakeClock{now: time.Now()})

	_, err := svc.Create(ctx, "   ", "")
	require.ErrorIs(t, err, model.ErrInvalid)
}
