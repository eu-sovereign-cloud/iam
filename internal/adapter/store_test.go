package adapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/eu-sovereign-cloud/iam/internal/adapter"
	"github.com/eu-sovereign-cloud/iam/internal/model"
)

func newTestStore(t *testing.T) *adapter.Store {
	t.Helper()
	client := k8sfake.NewClientset()
	store := adapter.NewStore(client, "iam-system")
	require.NoError(t, store.Load(context.Background()))
	return store
}

func TestUserCRUD(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	u := model.User{Subject: "alice@example.com", DisplayName: "Alice", Admin: true, CreatedAt: time.Now().UTC().Truncate(time.Second)}
	require.NoError(t, store.CreateUser(ctx, u))

	require.ErrorIs(t, store.CreateUser(ctx, u), model.ErrConflict)

	got, err := store.GetUser(ctx, u.Subject)
	require.NoError(t, err)
	require.Equal(t, u, got)

	list, err := store.ListUsers(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	got.Admin = false
	require.NoError(t, store.UpdateUser(ctx, got))
	got2, err := store.GetUser(ctx, u.Subject)
	require.NoError(t, err)
	require.False(t, got2.Admin)

	require.NoError(t, store.DeleteUser(ctx, u.Subject))
	_, err = store.GetUser(ctx, u.Subject)
	require.ErrorIs(t, err, model.ErrNotFound)
}

func TestTenantCRUD(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	tn := model.Tenant{TenantID: "tenant-1", DisplayName: "Tenant One", CreatedAt: time.Now().UTC().Truncate(time.Second)}
	require.NoError(t, store.CreateTenant(ctx, tn))
	require.ErrorIs(t, store.CreateTenant(ctx, tn), model.ErrConflict)

	got, err := store.GetTenant(ctx, tn.TenantID)
	require.NoError(t, err)
	require.Equal(t, tn, got)

	require.NoError(t, store.DeleteTenant(ctx, tn.TenantID))
	_, err = store.GetTenant(ctx, tn.TenantID)
	require.ErrorIs(t, err, model.ErrNotFound)
}

func TestGrantCRUD(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	g := model.Grant{Subject: "alice@example.com", TenantID: "tenant-1", GrantedAt: time.Now().UTC().Truncate(time.Second), GrantedBy: "admin"}
	require.NoError(t, store.CreateGrant(ctx, g))
	require.ErrorIs(t, store.CreateGrant(ctx, g), model.ErrConflict)

	list, err := store.ListGrantsBySubject(ctx, g.Subject)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, g, list[0])

	require.NoError(t, store.DeleteGrant(ctx, g.Subject, g.TenantID))
	list, err = store.ListGrantsBySubject(ctx, g.Subject)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestPATCRUD(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	exp := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	p := model.PAT{
		ID: "pat-1", Subject: "alice@example.com", Name: "laptop",
		Scope:     &model.TokenScope{Tenants: []string{"tenant-1"}},
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		ExpiresAt: exp,
	}
	require.NoError(t, store.CreatePAT(ctx, p))

	got, err := store.GetPAT(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, p, got)

	list, err := store.ListPATsBySubject(ctx, p.Subject)
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, store.DeletePAT(ctx, p.ID))
	_, err = store.GetPAT(ctx, p.ID)
	require.ErrorIs(t, err, model.ErrNotFound)
}

func TestLoadRoundTrip(t *testing.T) {
	ctx := context.Background()
	client := k8sfake.NewClientset()
	store := adapter.NewStore(client, "iam-system")
	require.NoError(t, store.Load(ctx))

	require.NoError(t, store.CreateUser(ctx, model.User{Subject: "bob", CreatedAt: time.Now().UTC().Truncate(time.Second)}))
	require.NoError(t, store.CreateTenant(ctx, model.Tenant{TenantID: "t1", CreatedAt: time.Now().UTC().Truncate(time.Second)}))

	// A fresh Store against the same fake client, as if iamd restarted,
	// must see everything the first Store wrote (ADR 0009).
	reloaded := adapter.NewStore(client, "iam-system")
	require.NoError(t, reloaded.Load(ctx))

	users, err := reloaded.ListUsers(ctx)
	require.NoError(t, err)
	require.Len(t, users, 1)

	tenants, err := reloaded.ListTenants(ctx)
	require.NoError(t, err)
	require.Len(t, tenants, 1)
}
