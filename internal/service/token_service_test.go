package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/service"
)

func TestTokenServiceExchange(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	users := newFakeUserStore()
	tenants := newFakeTenantStore()
	grants := newFakeGrantStore()
	pats := newFakePATStore()
	tokens := &fakeTokenGenerator{}

	require.NoError(t, users.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))
	require.NoError(t, tenants.CreateTenant(ctx, model.Tenant{TenantID: "tenant-1", CreatedAt: clock.now}))
	require.NoError(t, grants.CreateGrant(ctx, model.Grant{Subject: "alice", TenantID: "tenant-1", GrantedAt: clock.now}))

	patSvc := service.NewPATService(pats, tokens, clock)
	_, raw, err := patSvc.Create(ctx, "alice", "laptop", nil, 0)
	require.NoError(t, err)

	tokenSvc := service.NewTokenService(patSvc, users, grants, fakeSigner{}, clock, "https://iam.example.com", "ecp-gateway", 15*time.Minute)

	issued, err := tokenSvc.Exchange(ctx, raw)
	require.NoError(t, err)
	require.Equal(t, "Bearer", issued.TokenType)
	require.Equal(t, int64(15*60), issued.ExpiresIn)
	require.Contains(t, issued.AccessToken, "sub=alice")
	require.Contains(t, issued.AccessToken, "tenant-1")
}

func TestTokenServiceExchange_UnknownPAT(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	patSvc := service.NewPATService(newFakePATStore(), &fakeTokenGenerator{}, clock)
	tokenSvc := service.NewTokenService(patSvc, newFakeUserStore(), newFakeGrantStore(), fakeSigner{}, clock, "iss", "aud", time.Minute)

	_, err := tokenSvc.Exchange(ctx, "not-a-real-token")
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestTokenServiceExchange_RevokedPATFailsAfterDelete(t *testing.T) {
	ctx := context.Background()
	clock := fakeClock{now: time.Now()}
	users := newFakeUserStore()
	require.NoError(t, users.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: clock.now}))

	pats := newFakePATStore()
	tokens := &fakeTokenGenerator{}
	patSvc := service.NewPATService(pats, tokens, clock)
	tokenSvc := service.NewTokenService(patSvc, users, newFakeGrantStore(), fakeSigner{}, clock, "iss", "aud", time.Minute)

	p, raw, err := patSvc.Create(ctx, "alice", "laptop", nil, 0)
	require.NoError(t, err)

	_, err = tokenSvc.Exchange(ctx, raw)
	require.NoError(t, err)

	require.NoError(t, patSvc.Revoke(ctx, p.ID))

	_, err = tokenSvc.Exchange(ctx, raw)
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestTokenServiceExchange_ExpiredPAT(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	clockAtCreate := fakeClock{now: now}
	users := newFakeUserStore()
	require.NoError(t, users.CreateUser(ctx, model.User{Subject: "alice", CreatedAt: now}))

	pats := newFakePATStore()
	tokens := &fakeTokenGenerator{}
	createSvc := service.NewPATService(pats, tokens, clockAtCreate)
	_, raw, err := createSvc.Create(ctx, "alice", "laptop", nil, time.Minute)
	require.NoError(t, err)

	// Exchange as if an hour has passed since creation, well past the PAT's own 1-minute expiry.
	clockLater := fakeClock{now: now.Add(time.Hour)}
	exchangeSvc := service.NewPATService(pats, tokens, clockLater)
	tokenSvc := service.NewTokenService(exchangeSvc, users, newFakeGrantStore(), fakeSigner{}, clockLater, "iss", "aud", time.Minute)

	_, err = tokenSvc.Exchange(ctx, raw)
	require.ErrorIs(t, err, model.ErrForbidden)
}
