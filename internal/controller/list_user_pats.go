package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListUserPATs lists a subject's PATs (metadata only, never the signed
// JWT itself — that's only ever returned once, at creation). Self-or-admin.
type ListUserPATs struct {
	PATs ports.PATStore
}

func (c *ListUserPATs) Do(ctx context.Context, subject string) ([]model.PAT, error) {
	if err := model.RequireSelfOrAdmin(ctx, subject); err != nil {
		return nil, err
	}
	return c.PATs.ListPATsBySubject(ctx, subject)
}
