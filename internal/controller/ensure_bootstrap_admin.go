package controller

import (
	"context"
	"fmt"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// BootstrapAdminSubject is the subject of the admin User IAM creates on
// first startup if no admin User exists yet.
const BootstrapAdminSubject = "admin"

// bootstrapIdentitySubject names the synthetic caller injected into ctx
// for the duration of Do, so the admin-only ListUsers/CreateUser/CreatePAT
// it composes don't reject a startup call that has no real authenticated
// caller yet. Never persisted or looked up against a real User record.
const bootstrapIdentitySubject = "iamd-bootstrap"

// EnsureBootstrapAdmin creates the first admin User and issues it a PAT if
// no User with Admin: true exists yet (ADR 0006). The signed PAT is
// returned so the caller can log it once; per ADR 0012 it is never
// persisted, only its metadata (including its jti) is. Composes
// ListUsers, CreateUser and CreatePAT rather than duplicating their logic.
type EnsureBootstrapAdmin struct {
	ListUsers  *ListUsers
	CreateUser *CreateUser
	CreatePAT  *CreatePAT
}

func (c *EnsureBootstrapAdmin) Do(ctx context.Context) (rawPAT string, created bool, err error) {
	ctx = model.WithIdentity(ctx, model.User{Subject: bootstrapIdentitySubject, Admin: true})

	existing, err := c.ListUsers.Do(ctx)
	if err != nil {
		return "", false, fmt.Errorf("listing users: %w", err)
	}
	for _, u := range existing {
		if u.Admin {
			return "", false, nil
		}
	}

	if _, err := c.CreateUser.Do(ctx, BootstrapAdminSubject, "Bootstrap Administrator", true); err != nil {
		return "", false, fmt.Errorf("creating bootstrap admin user: %w", err)
	}

	_, raw, err := c.CreatePAT.Do(ctx, BootstrapAdminSubject, "bootstrap", nil, 0)
	if err != nil {
		return "", false, fmt.Errorf("creating bootstrap admin PAT: %w", err)
	}
	return raw, true, nil
}
