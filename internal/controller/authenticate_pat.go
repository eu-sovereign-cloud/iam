package controller

import (
	"context"
	"fmt"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// AuthenticatePAT resolves a raw bearer token (e.g. from an Authorization:
// Bearer header) to its PAT record: it verifies the token is a validly
// signed, unexpired JWT, then checks its jti is still known (not revoked).
type AuthenticatePAT struct {
	pats   ports.PATStore
	signer ports.Signer
	clock  ports.Clock
}

func NewAuthenticatePAT(pats ports.PATStore, signer ports.Signer, clock ports.Clock) *AuthenticatePAT {
	return &AuthenticatePAT{pats: pats, signer: signer, clock: clock}
}

func (c *AuthenticatePAT) Do(ctx context.Context, raw string) (model.PAT, error) {
	claims, err := c.signer.Verify(raw)
	if err != nil {
		return model.PAT{}, fmt.Errorf("%w: invalid token", model.ErrForbidden)
	}
	pat, err := c.pats.GetPAT(ctx, claims.ID)
	if err != nil {
		return model.PAT{}, fmt.Errorf("%w: unknown or revoked token", model.ErrForbidden)
	}
	if pat.Expired(c.clock.Now()) {
		return model.PAT{}, fmt.Errorf("%w: token expired", model.ErrForbidden)
	}
	return pat, nil
}
