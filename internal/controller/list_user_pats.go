package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListUserPATs lists a subject's PATs (metadata only, never the signed
// JWT itself — that's only ever returned once, at creation).
type ListUserPATs struct {
	PATs ports.PATStore
}

func (c *ListUserPATs) Do(ctx context.Context, subject string) ([]model.PAT, error) {
	return c.PATs.ListPATsBySubject(ctx, subject)
}
