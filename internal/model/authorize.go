package model

import (
	"context"
	"fmt"
)

// RequireAdmin returns ErrForbidden unless the caller in ctx is an admin.
func RequireAdmin(ctx context.Context) error {
	if !IdentityFromContext(ctx).Admin {
		return fmt.Errorf("%w: admin privileges required", ErrForbidden)
	}
	return nil
}

// RequireSelfOrAdmin returns ErrForbidden unless the caller in ctx is
// subject, or an admin.
func RequireSelfOrAdmin(ctx context.Context, subject string) error {
	caller := IdentityFromContext(ctx)
	if caller.Admin || caller.Subject == subject {
		return nil
	}
	return fmt.Errorf("%w: may only manage your own resources", ErrForbidden)
}
