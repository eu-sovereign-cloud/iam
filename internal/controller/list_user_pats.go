package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// ListUserPATs lists a subject's PATs (metadata only, never the signed
// JWT itself — that's only ever returned once, at creation).
type ListUserPATs struct {
	pats ports.PATStore
}

func NewListUserPATs(pats ports.PATStore) *ListUserPATs {
	return &ListUserPATs{pats: pats}
}

func (c *ListUserPATs) Do(ctx context.Context, subject string) ([]model.PAT, error) {
	return c.pats.ListPATsBySubject(ctx, subject)
}
