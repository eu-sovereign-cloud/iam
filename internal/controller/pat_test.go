package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/controller"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func TestCreatePAT_ListRevoke(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := fakeClock{now: now}
	grants := newFakeGrantStore()
	require.NoError(t, grants.CreateGrant(ctx, model.Grant{Subject: "alice", TenantID: "tenant-1", GrantedAt: now}))

	pats := newFakePATStore()
	create := &controller.CreatePAT{PATs: pats, Grants: grants, Signer: fakeSigner{}, Clock: clock, Issuer: "https://iam.example.com", Audience: []string{"ecp-gateway"}}
	list := &controller.ListUserPATs{PATs: pats}
	authenticate := &controller.AuthenticatePAT{PATs: pats, Signer: fakeSigner{}, Clock: clock}
	revoke := &controller.RevokePAT{PATs: pats}

	p, raw, err := create.Do(ctx, "alice", "laptop", nil, 0)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.True(t, p.ExpiresAt.After(now.Add(50*365*24*time.Hour)), "no ttl requested should mint a very long-lived token")

	claims, err := fakeSigner{}.Verify(raw)
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
	create := &controller.CreatePAT{PATs: newFakePATStore(), Grants: newFakeGrantStore(), Signer: fakeSigner{}, Clock: fakeClock{now: time.Now()}, Issuer: "iss", Audience: []string{"aud"}}
	_, _, err := create.Do(ctx, "", "name", nil, 0)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestAuthenticatePAT_UnknownToken(t *testing.T) {
	ctx := context.Background()
	authenticate := &controller.AuthenticatePAT{PATs: newFakePATStore(), Signer: fakeSigner{}, Clock: fakeClock{now: time.Now()}}
	_, err := authenticate.Do(ctx, "not-a-real-token")
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestCreatePAT_NameConflictPerSubject(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	create := &controller.CreatePAT{PATs: newFakePATStore(), Grants: newFakeGrantStore(), Signer: fakeSigner{}, Clock: clock, Issuer: "iss", Audience: []string{"aud"}}

	_, _, err := create.Do(ctx, "alice", "laptop", nil, 0)
	require.NoError(t, err)

	// Same subject, same name (even with incidental whitespace, which must
	// be trimmed before the conflict check — this was the exact bug
	// report): rejected.
	_, _, err = create.Do(ctx, "alice", "  laptop ", nil, 0)
	require.ErrorIs(t, err, model.ErrConflict)

	// A different subject may still use the same name.
	_, _, err = create.Do(ctx, "bob", "laptop", nil, 0)
	require.NoError(t, err)

	// Unnamed PATs never conflict with each other.
	_, _, err = create.Do(ctx, "alice", "", nil, 0)
	require.NoError(t, err)
	_, _, err = create.Do(ctx, "alice", "  ", nil, 0)
	require.NoError(t, err)
}
