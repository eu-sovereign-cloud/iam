package model

import "time"

// PAT is a Personal Access Token: the credential end users/tools exchange
// for a short-lived JWT (see TokenService). Only its hash is ever
// persisted; the raw secret is returned once, at creation time.
type PAT struct {
	ID        string
	Subject   string
	Name      string
	TokenHash string
	Scope     *TokenScope
	CreatedAt time.Time
	ExpiresAt *time.Time
}

// Expired reports whether the PAT itself (not any JWT minted from it) has
// passed its optional expiry.
func (p PAT) Expired(now time.Time) bool {
	return p.ExpiresAt != nil && now.After(*p.ExpiresAt)
}
