package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// BootstrapAdminSubject is the subject of the admin User IAM creates on
// first startup if no admin User exists yet.
const BootstrapAdminSubject = "admin"

// EnsureBootstrapAdmin creates the first admin User and a PAT for it if no
// User with Admin: true exists yet (ADR 0006). The raw PAT is returned so
// the caller can log it once; it is never persisted, only its hash is.
// Returns ok=false if an admin already exists and nothing was created.
func EnsureBootstrapAdmin(ctx context.Context, store *Store, tokens TokenGenerator, now time.Time) (rawPAT string, ok bool, err error) {
	users, err := store.ListUsers(ctx)
	if err != nil {
		return "", false, fmt.Errorf("listing users: %w", err)
	}
	for _, u := range users {
		if u.Admin {
			return "", false, nil
		}
	}

	admin := model.User{
		Subject:     BootstrapAdminSubject,
		DisplayName: "Bootstrap Administrator",
		Admin:       true,
		CreatedAt:   now,
	}
	if err := store.CreateUser(ctx, admin); err != nil {
		return "", false, fmt.Errorf("creating bootstrap admin user: %w", err)
	}

	raw, hash := tokens.NewToken()
	pat := model.PAT{
		ID:        uuid.NewString(),
		Subject:   admin.Subject,
		Name:      "bootstrap",
		TokenHash: hash,
		CreatedAt: now,
	}
	if err := store.CreatePAT(ctx, pat); err != nil {
		return "", false, fmt.Errorf("creating bootstrap admin PAT: %w", err)
	}
	return raw, true, nil
}
