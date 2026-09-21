package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestCreateUser_TrimsBeforeConflictCheck(t *testing.T) {
	ctx := context.Background()
	create := controller.NewCreateUser(newFakeUserStore(), fakeClock{now: time.Now()})

	_, err := create.Do(ctx, "alice@example.com", "Alice", false)
	require.NoError(t, err)

	_, err = create.Do(ctx, "  alice@example.com ", "Alice again", false)
	require.ErrorIs(t, err, model.ErrConflict)
}

func TestCreateUser_RequiresNonBlankSubject(t *testing.T) {
	ctx := context.Background()
	create := controller.NewCreateUser(newFakeUserStore(), fakeClock{now: time.Now()})

	_, err := create.Do(ctx, "   ", "", false)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestSetUserAdmin(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	create := controller.NewCreateUser(users, clock)
	setAdmin := controller.NewSetUserAdmin(users)

	u, err := create.Do(ctx, "alice", "Alice", false)
	require.NoError(t, err)
	require.False(t, u.Admin)

	u, err = setAdmin.Do(ctx, "alice", true)
	require.NoError(t, err)
	require.True(t, u.Admin)
}

func TestDeleteUser(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserStore()
	create := controller.NewCreateUser(users, fakeClock{now: time.Now()})
	get := controller.NewGetUser(users)
	deleteUser := controller.NewDeleteUser(users)

	_, err := create.Do(ctx, "alice", "", false)
	require.NoError(t, err)

	require.NoError(t, deleteUser.Do(ctx, "alice"))
	_, err = get.Do(ctx, "alice")
	require.ErrorIs(t, err, model.ErrNotFound)
}
