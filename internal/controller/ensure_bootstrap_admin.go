package controller

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
// persisted, only its metadata (including its jti) is. Composes
// ListUsers, CreateUser and CreatePAT rather than duplicating their logic.
type EnsureBootstrapAdmin struct {
	listUsers  *ListUsers
	createUser *CreateUser
	createPAT  *CreatePAT
}

func NewEnsureBootstrapAdmin(listUsers *ListUsers, createUser *CreateUser, createPAT *CreatePAT) *EnsureBootstrapAdmin {
	return &EnsureBootstrapAdmin{listUsers: listUsers, createUser: createUser, createPAT: createPAT}
}

func (c *EnsureBootstrapAdmin) Do(ctx context.Context) (rawPAT string, created bool, err error) {
	existing, err := c.listUsers.Do(ctx)
	if err != nil {
		return "", false, fmt.Errorf("listing users: %w", err)
	}
	for _, u := range existing {
		if u.Admin {
			return "", false, nil
		}
	}

	if _, err := c.createUser.Do(ctx, BootstrapAdminSubject, "Bootstrap Administrator", true); err != nil {
		return "", false, fmt.Errorf("creating bootstrap admin user: %w", err)
	}

	_, raw, err := c.createPAT.Do(ctx, BootstrapAdminSubject, "bootstrap", nil, 0)
	if err != nil {
		return "", false, fmt.Errorf("creating bootstrap admin PAT: %w", err)
	}
	return raw, true, nil
}
