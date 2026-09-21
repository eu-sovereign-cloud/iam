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
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateUser{Users: newFakeUserStore(), Clock: fakeClock{now: time.Now()}}

	_, err := create.Do(ctx, "alice@example.com", "Alice", false)
	require.NoError(t, err)

	_, err = create.Do(ctx, "  alice@example.com ", "Alice again", false)
	require.ErrorIs(t, err, model.ErrConflict)
}

func TestCreateUser_RequiresNonBlankSubject(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateUser{Users: newFakeUserStore(), Clock: fakeClock{now: time.Now()}}

	_, err := create.Do(ctx, "   ", "", false)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestSetUserAdmin(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	create := &controller.CreateUser{Users: users, Clock: clock}
	setAdmin := &controller.SetUserAdmin{Users: users}

	u, err := create.Do(ctx, "alice", "Alice", false)
	require.NoError(t, err)
	require.False(t, u.Admin)

	u, err = setAdmin.Do(ctx, "alice", true)
	require.NoError(t, err)
	require.True(t, u.Admin)
}

func TestDeleteUser(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	users := newFakeUserStore()
	create := &controller.CreateUser{Users: users, Clock: fakeClock{now: time.Now()}}
	get := &controller.GetUser{Users: users}
	deleteUser := &controller.DeleteUser{Users: users}

	_, err := create.Do(ctx, "alice", "", false)
	require.NoError(t, err)

	require.NoError(t, deleteUser.Do(ctx, "alice"))
	_, err = get.Do(ctx, "alice")
	require.ErrorIs(t, err, model.ErrNotFound)
}
