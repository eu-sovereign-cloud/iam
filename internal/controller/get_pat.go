package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// GetPAT fetches a single PAT's metadata by ID. Self-or-admin, checked
// against the PAT's actual owner once fetched (the caller only supplies an
// id, not a subject).
type GetPAT struct {
	PATs ports.PATStore
}

func (c *GetPAT) Do(ctx context.Context, id string) (model.PAT, error) {
	p, err := c.PATs.GetPAT(ctx, id)
	if err != nil {
		return model.PAT{}, err
	}
	if err := model.RequireSelfOrAdmin(ctx, p.Subject); err != nil {
		return model.PAT{}, err
	}
	return p, nil
}
