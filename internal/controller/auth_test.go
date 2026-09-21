package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestAuthenticateUser(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	pats := newFakePATStore()
	grants := newFakeGrantStore()

	require.NoError(t, users.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))

	createPAT := &controller.CreatePAT{PATs: pats, Grants: grants, Signer: fakeSigner{}, Clock: clock, Issuer: "iss", Audience: []string{"aud"}}
	authenticatePAT := &controller.AuthenticatePAT{PATs: pats, Signer: fakeSigner{}, Clock: clock}
	authenticateUser := &controller.AuthenticateUser{PATs: authenticatePAT, Users: users}

	selfCtx := model.WithIdentity(ctx, model.User{Subject: "alice"})
	_, raw, err := createPAT.Do(selfCtx, "alice", "laptop", nil, 0)
	require.NoError(t, err)

	u, err := authenticateUser.Do(ctx, raw)
	require.NoError(t, err)
	require.Equal(t, "alice", u.Subject)
}

func TestAuthenticateUser_UnknownSubject(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	pats := newFakePATStore()
	grants := newFakeGrantStore()

	// A PAT for a subject that was never registered as a User (e.g. the
	// User was deleted after the PAT was issued).
	createPAT := &controller.CreatePAT{PATs: pats, Grants: grants, Signer: fakeSigner{}, Clock: clock, Issuer: "iss", Audience: []string{"aud"}}
	authenticatePAT := &controller.AuthenticatePAT{PATs: pats, Signer: fakeSigner{}, Clock: clock}
	authenticateUser := &controller.AuthenticateUser{PATs: authenticatePAT, Users: newFakeUserStore()}

	selfCtx := model.WithIdentity(ctx, model.User{Subject: "ghost"})
	_, raw, err := createPAT.Do(selfCtx, "ghost", "laptop", nil, 0)
	require.NoError(t, err)

	_, err = authenticateUser.Do(ctx, raw)
	require.ErrorIs(t, err, model.ErrNotFound)
}
