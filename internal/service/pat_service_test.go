package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/service"
)

func TestPATServiceCreateListRevoke(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := fakeClock{now: now}
	grants := newFakeGrantStore()
	require.NoError(t, grants.CreateGrant(ctx, model.Grant{Subject: "alice", TenantID: "tenant-1", GrantedAt: now}))

	svc := service.NewPATService(newFakePATStore(), grants, fakeSigner{}, clock, "https://iam.example.com", "ecp-gateway")

	p, raw, err := svc.Create(ctx, "alice", "laptop", nil, 0)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.True(t, p.ExpiresAt.After(now.Add(50*365*24*time.Hour)), "no ttl requested should mint a very long-lived token")

	claims, err := fakeSigner{}.Verify(raw)
	require.NoError(t, err)
	require.Equal(t, "alice", claims.Subject)
	require.Equal(t, []string{"tenant-1"}, claims.Tenants)

	list, err := svc.ListBySubject(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, list, 1)

	_, err = svc.Authenticate(ctx, raw)
	require.NoError(t, err)

	require.NoError(t, svc.Revoke(ctx, p.ID))

	// The JWT itself is still validly signed and unexpired, but IAM's own
	// authentication now rejects it because its jti is no longer known
	// (ADR 0012's revocation gap is scoped to *other* verifiers, not IAM).
	_, err = svc.Authenticate(ctx, raw)
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestPATServiceCreate_RequiresSubject(t *testing.T) {
	ctx := context.Background()
	svc := service.NewPATService(newFakePATStore(), newFakeGrantStore(), fakeSigner{}, fakeClock{now: time.Now()}, "iss", "aud")
	_, _, err := svc.Create(ctx, "", "name", nil, 0)
	require.ErrorIs(t, err, model.ErrInvalid)
}

func TestPATServiceAuthenticate_UnknownToken(t *testing.T) {
	ctx := context.Background()
	svc := service.NewPATService(newFakePATStore(), newFakeGrantStore(), fakeSigner{}, fakeClock{now: time.Now()}, "iss", "aud")
	_, err := svc.Authenticate(ctx, "not-a-real-token")
	require.ErrorIs(t, err, model.ErrForbidden)
}
