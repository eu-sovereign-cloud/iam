package service

import (
	"context"
	"fmt"
)

// BootstrapAdminSubject is the subject of the admin User IAM creates on
// first startup if no admin User exists yet.
const BootstrapAdminSubject = "admin"

// EnsureBootstrapAdmin creates the first admin User and issues it a PAT if
// no User with Admin: true exists yet (ADR 0006). The signed PAT is
// returned so the caller can log it once; per ADR 0012 it is never
// persisted, only its metadata (including its jti) is.
//
// This lives in the service layer, not the adapter layer, because issuing
// a PAT now means building and signing real claims (PATService.Create) —
// business logic that orchestrates UserService and PATService, not
// something the adapter layer should duplicate.
func EnsureBootstrapAdmin(ctx context.Context, users *UserService, pats *PATService) (rawPAT string, created bool, err error) {
	existing, err := users.List(ctx)
	if err != nil {
		return "", false, fmt.Errorf("listing users: %w", err)
	}
	for _, u := range existing {
		if u.Admin {
			return "", false, nil
		}
	}

	if _, err := users.Create(ctx, BootstrapAdminSubject, "Bootstrap Administrator", true); err != nil {
		return "", false, fmt.Errorf("creating bootstrap admin user: %w", err)
	}

	_, raw, err := pats.Create(ctx, BootstrapAdminSubject, "bootstrap", nil, 0)
	if err != nil {
		return "", false, fmt.Errorf("creating bootstrap admin PAT: %w", err)
	}
	return raw, true, nil
}
