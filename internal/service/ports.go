// Package service implements IAM's use cases and declares the ports
// (interfaces) it needs from the adapter layer.
package service

import (
	"context"
	"time"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// UserStore is the persistence port for Users.
type UserStore interface {
	CreateUser(ctx context.Context, u model.User) error
	GetUser(ctx context.Context, subject string) (model.User, error)
	ListUsers(ctx context.Context) ([]model.User, error)
	UpdateUser(ctx context.Context, u model.User) error
	DeleteUser(ctx context.Context, subject string) error
}

// TenantStore is the persistence port for Tenants.
type TenantStore interface {
	CreateTenant(ctx context.Context, t model.Tenant) error
	GetTenant(ctx context.Context, tenantID string) (model.Tenant, error)
	ListTenants(ctx context.Context) ([]model.Tenant, error)
	DeleteTenant(ctx context.Context, tenantID string) error
}

// GrantStore is the persistence port for Grants.
type GrantStore interface {
	CreateGrant(ctx context.Context, g model.Grant) error
	ListGrantsBySubject(ctx context.Context, subject string) ([]model.Grant, error)
	DeleteGrant(ctx context.Context, subject, tenantID string) error
}

// PATStore is the persistence port for PATs.
type PATStore interface {
	CreatePAT(ctx context.Context, p model.PAT) error
	GetPAT(ctx context.Context, id string) (model.PAT, error)
	ListPATsBySubject(ctx context.Context, subject string) ([]model.PAT, error)
	DeletePAT(ctx context.Context, id string) error
}

// Signer mints and verifies the signed JWTs that PATs are (ADR 0012).
type Signer interface {
	Sign(claims model.Claims) (string, error)
	Verify(token string) (model.Claims, error)
}

// Clock is injected so services are deterministic in tests.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock backed by time.Now.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
