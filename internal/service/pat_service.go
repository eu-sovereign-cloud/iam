package service

import (
	"context"
	"fmt"
	"time"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// PATService manages Personal Access Tokens. It is self-service: the
// controller layer lets a User act on their own subject regardless of the
// admin flag, and lets admins act on any subject.
type PATService struct {
	store  PATStore
	tokens TokenGenerator
	clock  Clock
}

func NewPATService(store PATStore, tokens TokenGenerator, clock Clock) *PATService {
	return &PATService{store: store, tokens: tokens, clock: clock}
}

// Create mints a new PAT for subject and returns the domain record plus the
// raw secret, which is never persisted and is only available this once.
func (s *PATService) Create(ctx context.Context, subject, name string, scope *model.TokenScope, ttl time.Duration) (model.PAT, string, error) {
	if subject == "" {
		return model.PAT{}, "", fmt.Errorf("%w: subject is required", model.ErrInvalid)
	}
	raw, hash := s.tokens.NewToken()
	now := s.clock.Now()
	p := model.PAT{
		ID:        newID(),
		Subject:   subject,
		Name:      name,
		TokenHash: hash,
		Scope:     scope,
		CreatedAt: now,
	}
	if ttl > 0 {
		exp := now.Add(ttl)
		p.ExpiresAt = &exp
	}
	if err := s.store.CreatePAT(ctx, p); err != nil {
		return model.PAT{}, "", err
	}
	return p, raw, nil
}

func (s *PATService) ListBySubject(ctx context.Context, subject string) ([]model.PAT, error) {
	return s.store.ListPATsBySubject(ctx, subject)
}

func (s *PATService) Get(ctx context.Context, id string) (model.PAT, error) {
	return s.store.GetPAT(ctx, id)
}

// Revoke deletes the PAT's Secret outright (ADR 0004): this immediately
// blocks future token exchanges. It does not retroactively invalidate any
// JWT already issued from it before its (short) expiry.
func (s *PATService) Revoke(ctx context.Context, id string) error {
	return s.store.DeletePAT(ctx, id)
}

// Authenticate resolves a raw PAT secret (e.g. from an Authorization:
// Bearer header) to its PAT record, rejecting unknown, revoked or expired
// tokens.
func (s *PATService) Authenticate(ctx context.Context, raw string) (model.PAT, error) {
	hash := s.tokens.Hash(raw)
	pat, err := s.store.GetPATByHash(ctx, hash)
	if err != nil {
		return model.PAT{}, fmt.Errorf("%w: unknown or revoked token", model.ErrForbidden)
	}
	if pat.Expired(s.clock.Now()) {
		return model.PAT{}, fmt.Errorf("%w: token expired", model.ErrForbidden)
	}
	return pat, nil
}
