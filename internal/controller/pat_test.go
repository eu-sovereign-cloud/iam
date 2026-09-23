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

func TestCreatePAT_ListRevoke(t *testing.T) {
	ctx := model.WithIdentity(context.Background(), model.User{Subject: "alice"})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := fakeClock{now: now}
	store := memorystore.New()
	require.NoError(t, store.CreateGrant(ctx, model.Grant{Subject: "alice", TenantID: "tenant-1", GrantedAt: now}))

	create := &controller.CreatePAT{PATs: store, Grants: store, Signer: memorycrypto.Signer{}, Clock: clock, Issuer: "https://iam.example.com", Audience: []string{"ecp-gateway"}}
	list := &controller.ListUserPATs{PATs: store}
	authenticate := &controller.AuthenticatePAT{PATs: store, Signer: memorycrypto.Signer{}, Clock: clock}
	revoke := &controller.RevokePAT{PATs: store}

	p, raw, err := create.Do(ctx, "alice", "laptop", nil, 0)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.True(t, p.ExpiresAt.After(now.Add(50*365*24*time.Hour)), "no ttl requested should mint a very long-lived token")

	claims, err := (memorycrypto.Signer{}).Verify(raw)
	require.NoError(t, err)
	require.Equal(t, "alice", claims.Subject)
	require.Equal(t, []string{"tenant-1"}, claims.Tenants)

	got, err := list.Do(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, got, 1)

	_, err = authenticate.Do(ctx, raw)
	require.NoError(t, err)

	require.NoError(t, revoke.Do(ctx, p.ID))

	// The JWT itself is still validly signed and unexpired, but IAM's own
	// authentication now rejects it because its jti is no longer known
	// (ADR 0012's revocation gap is scoped to *other* verifiers, not IAM).
	_, err = authenticate.Do(ctx, raw)
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestCreatePAT_RequiresSubject(t *testing.T) {
	ctx := context.Background()
	create := &controller.CreatePAT{PATs: memorystore.New(), Grants: memorystore.New(), Signer: memorycrypto.Signer{}, Clock: fakeClock{now: time.Now()}, Issuer: "iss", Audience: []string{"aud"}}
	_, _, err := create.Do(ctx, "", "name", nil, 0)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestAuthenticatePAT_UnknownToken(t *testing.T) {
	ctx := context.Background()
	authenticate := &controller.AuthenticatePAT{PATs: memorystore.New(), Signer: memorycrypto.Signer{}, Clock: fakeClock{now: time.Now()}}
	_, err := authenticate.Do(ctx, "not-a-real-token")
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestCreatePAT_NameConflictPerSubject(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	clock := fakeClock{now: time.Now()}
	create := &controller.CreatePAT{PATs: memorystore.New(), Grants: memorystore.New(), Signer: memorycrypto.Signer{}, Clock: clock, Issuer: "iss", Audience: []string{"aud"}}

	_, _, err := create.Do(adminCtx, "alice", "laptop", nil, 0)
	require.NoError(t, err)

	// Same subject, same name (even with incidental whitespace, which must
	// be trimmed before the conflict check — this was the exact bug
	// report): rejected.
	_, _, err = create.Do(adminCtx, "alice", "  laptop ", nil, 0)
	require.ErrorIs(t, err, model.ErrConflict)

	// A different subject may still use the same name.
	_, _, err = create.Do(adminCtx, "bob", "laptop", nil, 0)
	require.NoError(t, err)
}

func TestCreatePAT_RequiresName(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreatePAT{PATs: memorystore.New(), Grants: memorystore.New(), Signer: memorycrypto.Signer{}, Clock: fakeClock{now: time.Now()}, Issuer: "iss", Audience: []string{"aud"}}

	_, _, err := create.Do(adminCtx, "alice", "", nil, 0)
	require.ErrorIs(t, err, model.ErrInvalid)

	// Whitespace-only doesn't count either.
	_, _, err = create.Do(adminCtx, "alice", "   ", nil, 0)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestCreatePAT_RequiresDNS1123Name(t *testing.T) {
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreatePAT{PATs: memorystore.New(), Grants: memorystore.New(), Signer: memorycrypto.Signer{}, Clock: fakeClock{now: time.Now()}, Issuer: "iss", Audience: []string{"aud"}}

	_, _, err := create.Do(adminCtx, "alice", "My Laptop", nil, 0)
	require.ErrorIs(t, err, model.ErrInvalid)

	_, _, err = create.Do(adminCtx, "alice", "my-laptop", nil, 0)
	require.NoError(t, err, "a valid DNS-1123 label must still be accepted")
}

func TestCreatePAT_SelfOrAdmin(t *testing.T) {
	create := &controller.CreatePAT{PATs: memorystore.New(), Grants: memorystore.New(), Signer: memorycrypto.Signer{}, Clock: fakeClock{now: time.Now()}, Issuer: "iss", Audience: []string{"aud"}}

	otherCtx := model.WithIdentity(context.Background(), model.User{Subject: "bob"})
	_, _, err := create.Do(otherCtx, "alice", "laptop", nil, 0)
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestRevokePAT_SelfOrAdmin(t *testing.T) {
	store := memorystore.New()
	adminCtx := model.WithIdentity(context.Background(), model.User{Subject: "admin", Admin: true})
	create := &controller.CreatePAT{PATs: store, Grants: memorystore.New(), Signer: memorycrypto.Signer{}, Clock: fakeClock{now: time.Now()}, Issuer: "iss", Audience: []string{"aud"}}
	p, _, err := create.Do(adminCtx, "alice", "laptop", nil, 0)
	require.NoError(t, err)

	revoke := &controller.RevokePAT{PATs: store}
	otherCtx := model.WithIdentity(context.Background(), model.User{Subject: "bob"})
	require.ErrorIs(t, revoke.Do(otherCtx, p.ID), model.ErrForbidden)

	require.NoError(t, revoke.Do(adminCtx, p.ID))
}
