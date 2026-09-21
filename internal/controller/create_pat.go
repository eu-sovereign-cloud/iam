package controller

import (
	"context"
	"fmt"
	"strings"
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

// CreatePAT mints a new PAT for subject: a signed JWT (sub=subject,
// tenants=the subject's current Grants resolved right now) plus a
// metadata record of it. The signed JWT is returned once and never
// persisted (ADR 0012) — only its jti and bookkeeping metadata are. It is
// self-service: callers let a User act on their own subject regardless of
// the admin flag, and let admins act on any subject.
type CreatePAT struct {
	PATs     ports.PATStore
	Grants   ports.GrantStore
	Signer   ports.Signer
	Clock    ports.Clock
	Issuer   string
	Audience []string
}

func (c *CreatePAT) Do(ctx context.Context, subject, name string, scope *model.TokenScope, ttl time.Duration) (model.PAT, string, error) {
	subject = strings.TrimSpace(subject)
	name = strings.TrimSpace(name)
	if subject == "" {
		return model.PAT{}, "", fmt.Errorf("%w: subject is required", model.ErrInvalid)
	}
	if ttl <= 0 {
		ttl = noExpiryDuration
	}

	grants, err := c.Grants.ListGrantsBySubject(ctx, subject)
	if err != nil {
		return model.PAT{}, "", err
	}
	tenants := make([]string, 0, len(grants))
	for _, g := range grants {
		tenants = append(tenants, g.TenantID)
	}

	now := c.Clock.Now()
	exp := now.Add(ttl)
	jti := newID()

	claims := model.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    c.Issuer,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
		Scope:   scope,
		Tenants: tenants,
	}
	if len(c.Audience) > 0 {
		claims.Audience = jwt.ClaimStrings(c.Audience)
	}

	signed, err := c.Signer.Sign(claims)
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
	if err := c.PATs.CreatePAT(ctx, p); err != nil {
		return model.PAT{}, "", err
	}
	return p, signed, nil
}
