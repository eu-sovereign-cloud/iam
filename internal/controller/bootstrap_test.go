package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestEnsureBootstrapAdmin_CreatesOnlyOnce(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	pats := newFakePATStore()
	grants := newFakeGrantStore()

	listUsers := &controller.ListUsers{Users: users}
	createUser := &controller.CreateUser{Users: users, Clock: clock}
	createPAT := &controller.CreatePAT{PATs: pats, Grants: grants, Signer: fakeSigner{}, Clock: clock, Issuer: "iss", Audience: []string{"aud"}}
	ensure := &controller.EnsureBootstrapAdmin{ListUsers: listUsers, CreateUser: createUser, CreatePAT: createPAT}

	raw, created, err := ensure.Do(ctx)
	require.NoError(t, err)
	require.True(t, created)
	require.NotEmpty(t, raw)

	adminCtx := model.WithIdentity(ctx, model.User{Subject: "admin", Admin: true})
	admin, err := (&controller.GetUser{Users: users}).Do(adminCtx, controller.BootstrapAdminSubject)
	require.NoError(t, err)
	require.True(t, admin.Admin)

	// A second call must not mint a second admin/PAT now that one exists.
	raw2, created2, err := ensure.Do(ctx)
	require.NoError(t, err)
	require.False(t, created2)
	require.Empty(t, raw2)
}
