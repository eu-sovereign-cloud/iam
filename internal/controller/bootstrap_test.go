package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
)

func TestEnsureBootstrapAdmin_CreatesOnlyOnce(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	pats := newFakePATStore()
	grants := newFakeGrantStore()

	listUsers := controller.NewListUsers(users)
	createUser := controller.NewCreateUser(users, clock)
	createPAT := controller.NewCreatePAT(pats, grants, fakeSigner{}, clock, "iss", "aud")
	ensure := controller.NewEnsureBootstrapAdmin(listUsers, createUser, createPAT)

	raw, created, err := ensure.Do(ctx)
	require.NoError(t, err)
	require.True(t, created)
	require.NotEmpty(t, raw)

	admin, err := controller.NewGetUser(users).Do(ctx, controller.BootstrapAdminSubject)
	require.NoError(t, err)
	require.True(t, admin.Admin)

	// A second call must not mint a second admin/PAT now that one exists.
	raw2, created2, err := ensure.Do(ctx)
	require.NoError(t, err)
	require.False(t, created2)
	require.Empty(t, raw2)
}
