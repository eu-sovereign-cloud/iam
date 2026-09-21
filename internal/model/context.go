package model

import "context"

type identityContextKey struct{}

// WithIdentity returns a copy of ctx carrying u as the authenticated
// caller. Both driving adapters (internal/service's REST middleware,
// internal/web's cookie-based auth) use this same helper rather than each
// keeping their own copy.
func WithIdentity(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, identityContextKey{}, u)
}

// IdentityFromContext returns the authenticated caller set by
// WithIdentity. It panics if called on a request that hasn't been through
// an auth check first — a routing bug, not a runtime condition to handle
// gracefully, since both driving adapters always wrap handlers that need
// it in their own require-auth middleware.
func IdentityFromContext(ctx context.Context) User {
	u, ok := ctx.Value(identityContextKey{}).(User)
	if !ok {
		panic("model: no identity in context; handler not wrapped in a require-auth middleware")
	}
	return u
}
