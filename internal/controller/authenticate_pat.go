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
	PATs   ports.PATStore
	Signer ports.Signer
	Clock  ports.Clock
}

func (c *AuthenticatePAT) Do(ctx context.Context, raw string) (model.PAT, error) {
	claims, err := c.Signer.Verify(raw)
	if err != nil {
		return model.PAT{}, fmt.Errorf("%w: invalid token", model.ErrForbidden)
	}
	pat, err := c.PATs.GetPAT(ctx, claims.ID)
	if err != nil {
		return model.PAT{}, fmt.Errorf("%w: unknown or revoked token", model.ErrForbidden)
	}
	if pat.Expired(c.Clock.Now()) {
		return model.PAT{}, fmt.Errorf("%w: token expired", model.ErrForbidden)
	}
	return pat, nil
}
