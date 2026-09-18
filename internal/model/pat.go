package model

import "time"

// PAT is a Personal Access Token: a signed JWT (see Claims) issued
// directly to a user as their bearer credential — there is no separate
// exchange step (ADR 0012). IAM never persists the raw JWT, only metadata
// about it (enough to list it and to check its jti hasn't been revoked);
// the signed string itself is only ever shown once, at creation time.
type PAT struct {
	ID        string // the JWT's jti
	Subject   string
	Name      string
	Scope     *TokenScope
	CreatedAt time.Time
	ExpiresAt time.Time // always set: a JWT's exp is mandatory (ADR 0012)
}

// Expired reports whether the PAT has passed its expiry.
func (p PAT) Expired(now time.Time) bool {
	return now.After(p.ExpiresAt)
}
