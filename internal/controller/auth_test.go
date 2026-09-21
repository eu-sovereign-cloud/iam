package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/adapter/memorycrypto"
	"github.com/eu-sovereign-cloud/iam/internal/adapter/memorystore"
	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestAuthenticateUser(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	store := memorystore.New()

	require.NoError(t, store.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))

	createPAT := &controller.CreatePAT{PATs: store, Grants: store, Signer: memorycrypto.Signer{}, Clock: clock, Issuer: "iss", Audience: []string{"aud"}}
	authenticatePAT := &controller.AuthenticatePAT{PATs: store, Signer: memorycrypto.Signer{}, Clock: clock}
	authenticateUser := &controller.AuthenticateUser{PATs: authenticatePAT, Users: store}

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
	store := memorystore.New()

	// A PAT for a subject that was never registered as a User (e.g. the
	// User was deleted after the PAT was issued).
	createPAT := &controller.CreatePAT{PATs: store, Grants: store, Signer: memorycrypto.Signer{}, Clock: clock, Issuer: "iss", Audience: []string{"aud"}}
	authenticatePAT := &controller.AuthenticatePAT{PATs: store, Signer: memorycrypto.Signer{}, Clock: clock}
	authenticateUser := &controller.AuthenticateUser{PATs: authenticatePAT, Users: memorystore.New()}

	selfCtx := model.WithIdentity(ctx, model.User{Subject: "ghost"})
	_, raw, err := createPAT.Do(selfCtx, "ghost", "laptop", nil, 0)
	require.NoError(t, err)

	_, err = authenticateUser.Do(ctx, raw)
	require.ErrorIs(t, err, model.ErrNotFound)
}
