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
	clock := fakeClock{now: time.Now()}
	svc := service.NewPATService(newFakePATStore(), &fakeTokenGenerator{}, clock)

	p, raw, err := svc.Create(ctx, "alice", "laptop", nil, 0)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.Nil(t, p.ExpiresAt)

	list, err := svc.ListBySubject(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, list, 1)

	_, err = svc.Authenticate(ctx, raw)
	require.NoError(t, err)

	require.NoError(t, svc.Revoke(ctx, p.ID))

	_, err = svc.Authenticate(ctx, raw)
	require.ErrorIs(t, err, model.ErrForbidden)
}

func TestPATServiceCreate_RequiresSubject(t *testing.T) {
	ctx := context.Background()
	svc := service.NewPATService(newFakePATStore(), &fakeTokenGenerator{}, fakeClock{now: time.Now()})
	_, _, err := svc.Create(ctx, "", "name", nil, 0)
	require.ErrorIs(t, err, model.ErrInvalid)
}
