package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type identityContextKey struct{}

func withIdentity(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, identityContextKey{}, u)
}

// identityFromContext returns the authenticated caller set by RequireAuth.
// It panics if called on a request that hasn't passed through RequireAuth,
// which would be a routing bug, not a runtime condition to handle
// gracefully.
func identityFromContext(ctx context.Context) model.User {
	u, ok := ctx.Value(identityContextKey{}).(model.User)
	if !ok {
		panic("controller: no identity in context; handler not wrapped in RequireAuth")
	}
	return u
}
