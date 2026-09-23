package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/adapter/memorystore"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestCreateUser_TrimsBeforeConflictCheck(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateUser{Users: memorystore.New(), Clock: fakeClock{now: time.Now()}}

	_, err := create.Do(ctx, "alice@example.com", "Alice", false)
	require.NoError(t, err)

	_, err = create.Do(ctx, "  alice@example.com ", "Alice again", false)
	require.ErrorIs(t, err, model.ErrConflict)
}

func TestCreateUser_RequiresNonBlankSubject(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateUser{Users: memorystore.New(), Clock: fakeClock{now: time.Now()}}

	_, err := create.Do(ctx, "   ", "", false)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestCreateUser_ValidatesSubjectCharset(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreateUser{Users: memorystore.New(), Clock: fakeClock{now: time.Now()}}

	_, err := create.Do(ctx, "alice <script>", "Alice", false)
	require.ErrorIs(t, err, model.ErrInvalid)

	// Both a plain identifier and an email-style subject must still be
	// accepted - Subject deliberately isn't DNS-1123 (that would forbid
	// '@' and reject every email-style subject this app otherwise uses).
	_, err = create.Do(ctx, "admin2", "", false)
	require.NoError(t, err)
	_, err = create.Do(ctx, "alice@example.com", "Alice", false)
	require.NoError(t, err)
}

func TestSetUserAdmin(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()
	create := &controller.CreateUser{Users: store, Clock: clock}
	setAdmin := &controller.SetUserAdmin{Users: store}

	u, err := create.Do(ctx, "alice", "Alice", false)
	require.NoError(t, err)
	require.False(t, u.Admin)

	u, err = setAdmin.Do(ctx, "alice", true)
	require.NoError(t, err)
	require.True(t, u.Admin)
}

func TestDeleteUser(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	store := memorystore.New()
	create := &controller.CreateUser{Users: store, Clock: fakeClock{now: time.Now()}}
	get := &controller.GetUser{Users: store}
	deleteUser := &controller.DeleteUser{Users: store}

	_, err := create.Do(ctx, "alice", "", false)
	require.NoError(t, err)

	require.NoError(t, deleteUser.Do(ctx, "alice"))
	_, err = get.Do(ctx, "alice")
	require.ErrorIs(t, err, model.ErrNotFound)
}
