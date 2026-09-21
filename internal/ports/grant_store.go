package ports

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// GrantStore is the persistence port for Grants.
type GrantStore interface {
	CreateGrant(ctx context.Context, g model.Grant) error
	GetGrant(ctx context.Context, subject, tenantID string) (model.Grant, error)
	ListGrantsBySubject(ctx context.Context, subject string) ([]model.Grant, error)
	UpdateGrant(ctx context.Context, g model.Grant) error
	DeleteGrant(ctx context.Context, subject, tenantID string) error
}
