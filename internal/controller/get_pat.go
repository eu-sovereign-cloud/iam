package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// GetPAT fetches a single PAT's metadata by ID.
type GetPAT struct {
	PATs ports.PATStore
}

func (c *GetPAT) Do(ctx context.Context, id string) (model.PAT, error) {
	return c.PATs.GetPAT(ctx, id)
}
