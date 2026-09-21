package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// noExpiryDuration is minted as the JWT's exp when a PAT is created with no
// requested TTL. A JWT's exp is mandatory for ecp's gateway to accept it
// (jwt.WithExpirationRequired), so "no expiry" means "effectively never,"
// not "absent" (ADR 0012).
const noExpiryDuration = 100 * 365 * 24 * time.Hour

// PATService issues and manages PATs. Per ADR 0012 a PAT *is* the signed
// JWT a caller uses as their bearer credential everywhere — there is no
// separate exchange step. It is self-service: the controller layer lets a
// User act on their own subject regardless of the admin flag, and lets
// admins act on any subject.
type PATService struct {
	store    ports.PATStore
	grants   ports.GrantStore
	signer   ports.Signer
	clock    ports.Clock
	issuer   string
	audience string
}

func NewPATService(store ports.PATStore, grants ports.GrantStore, signer ports.Signer, clock ports.Clock, issuer, audience string) *PATService {
	return &PATService{store: store, grants: grants, signer: signer, clock: clock, issuer: issuer, audience: audience}
}

// Create mints a new PAT for subject: a signed JWT (sub=subject, tenants=
// the subject's current Grants resolved right now) plus a metadata record
// of it. The signed JWT is returned once and never persisted (ADR 0012) —
// only its jti and bookkeeping metadata are.
func (s *PATService) Create(ctx context.Context, subject, name string, scope *model.TokenScope, ttl time.Duration) (model.PAT, string, error) {
	if subject == "" {
		return model.PAT{}, "", fmt.Errorf("%w: subject is required", model.ErrInvalid)
	}
	if ttl <= 0 {
		ttl = noExpiryDuration
	}

	grants, err := s.grants.ListGrantsBySubject(ctx, subject)
	if err != nil {
		return model.PAT{}, "", err
	}
	tenants := make([]string, 0, len(grants))
	for _, g := range grants {
		tenants = append(tenants, g.TenantID)
	}

	now := s.clock.Now()
	exp := now.Add(ttl)
	jti := newID()

	claims := model.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    s.issuer,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
		Scope:   scope,
		Tenants: tenants,
	}
	if s.audience != "" {
		claims.Audience = jwt.ClaimStrings{s.audience}
	}

	signed, err := s.signer.Sign(claims)
	if err != nil {
		return model.PAT{}, "", fmt.Errorf("signing PAT: %w", err)
	}

	p := model.PAT{
		ID:        jti,
		Subject:   subject,
		Name:      name,
		Scope:     scope,
		CreatedAt: now,
		ExpiresAt: exp,
	}
	if err := s.store.CreatePAT(ctx, p); err != nil {
		return model.PAT{}, "", err
	}
	return p, signed, nil
}

func (s *PATService) ListBySubject(ctx context.Context, subject string) ([]model.PAT, error) {
	return s.store.ListPATsBySubject(ctx, subject)
}

func (s *PATService) Get(ctx context.Context, id string) (model.PAT, error) {
	return s.store.GetPAT(ctx, id)
}

// Revoke deletes the PAT's metadata record outright (ADR 0004, ADR 0012):
// this immediately blocks the token from authenticating against IAM
// itself. It cannot invalidate a copy of the same JWT already being used
// directly against ecp's gateway, which verifies signature+expiry offline
// with no callback to IAM — an explicitly accepted trade-off (ADR 0012)
// until issue #2's /userinfo liveness check exists.
func (s *PATService) Revoke(ctx context.Context, id string) error {
	return s.store.DeletePAT(ctx, id)
}

// Authenticate resolves a raw bearer token (e.g. from an Authorization:
// Bearer header) to its PAT record: it verifies the token is a validly
// signed, unexpired JWT, then checks its jti is still known (not revoked).
func (s *PATService) Authenticate(ctx context.Context, raw string) (model.PAT, error) {
	claims, err := s.signer.Verify(raw)
	if err != nil {
		return model.PAT{}, fmt.Errorf("%w: invalid token", model.ErrForbidden)
	}
	pat, err := s.store.GetPAT(ctx, claims.ID)
	if err != nil {
		return model.PAT{}, fmt.Errorf("%w: unknown or revoked token", model.ErrForbidden)
	}
	if pat.Expired(s.clock.Now()) {
		return model.PAT{}, fmt.Errorf("%w: token expired", model.ErrForbidden)
	}
	return pat, nil
}
