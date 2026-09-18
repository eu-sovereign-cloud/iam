package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// TokenService exchanges a PAT for a short-lived JWT compatible with ecp's
// gateway auth middleware (ADR 0003).
type TokenService struct {
	pats     *PATService
	users    UserStore
	grants   GrantStore
	signer   Signer
	clock    Clock
	issuer   string
	audience string
	ttl      time.Duration
}

func NewTokenService(pats *PATService, users UserStore, grants GrantStore, signer Signer, clock Clock, issuer, audience string, ttl time.Duration) *TokenService {
	return &TokenService{
		pats: pats, users: users, grants: grants, signer: signer, clock: clock,
		issuer: issuer, audience: audience, ttl: ttl,
	}
}

// IssuedToken is the response returned to a successful exchange.
type IssuedToken struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
}

// Exchange looks up rawPAT by its hash, verifies it is neither expired nor
// revoked (revocation == the Secret no longer existing, ADR 0004), resolves
// the owning subject's current Grants into the "tenants" claim, and mints a
// JWT.
func (s *TokenService) Exchange(ctx context.Context, rawPAT string) (IssuedToken, error) {
	pat, err := s.pats.Authenticate(ctx, rawPAT)
	if err != nil {
		return IssuedToken{}, err
	}
	now := s.clock.Now()
	if _, err := s.users.GetUser(ctx, pat.Subject); err != nil {
		return IssuedToken{}, fmt.Errorf("%w: unknown subject", model.ErrForbidden)
	}
	grants, err := s.grants.ListGrantsBySubject(ctx, pat.Subject)
	if err != nil {
		return IssuedToken{}, err
	}
	tenants := make([]string, 0, len(grants))
	for _, g := range grants {
		tenants = append(tenants, g.TenantID)
	}

	exp := now.Add(s.ttl)
	claims := model.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   pat.Subject,
			Issuer:    s.issuer,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        newID(),
		},
		Scope:   pat.Scope,
		Tenants: tenants,
	}
	if s.audience != "" {
		claims.Audience = jwt.ClaimStrings{s.audience}
	}

	signed, err := s.signer.Sign(claims)
	if err != nil {
		return IssuedToken{}, fmt.Errorf("signing token: %w", err)
	}
	return IssuedToken{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.ttl.Seconds()),
	}, nil
}
