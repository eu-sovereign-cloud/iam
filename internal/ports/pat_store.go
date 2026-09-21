package ports

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// PATStore is the persistence port for PATs.
type PATStore interface {
	CreatePAT(ctx context.Context, p model.PAT) error
	GetPAT(ctx context.Context, id string) (model.PAT, error)
	ListPATsBySubject(ctx context.Context, subject string) ([]model.PAT, error)
	DeletePAT(ctx context.Context, id string) error
}
