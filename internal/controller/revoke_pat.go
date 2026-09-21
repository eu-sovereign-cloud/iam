package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// RevokePAT deletes a PAT's metadata record outright (ADR 0004, ADR 0012):
// this immediately blocks the token from authenticating against IAM
// itself. It cannot invalidate a copy of the same JWT already being used
// directly against ecp's gateway, which verifies signature+expiry offline
// with no callback to IAM — an explicitly accepted trade-off (ADR 0012)
// until issue #2's /userinfo liveness check exists.
type RevokePAT struct {
	PATs ports.PATStore
}

func (c *RevokePAT) Do(ctx context.Context, id string) error {
	return c.PATs.DeletePAT(ctx, id)
}
