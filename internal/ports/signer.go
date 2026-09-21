package ports

import (
	"crypto/ecdsa"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// Signer mints and verifies the signed JWTs that PATs are (ADR 0012), and
// exposes the public half of its signing key so it can be published as a
// JWKS (issue #2, ADR 0019).
type Signer interface {
	Sign(claims model.Claims) (string, error)
	Verify(token string) (model.Claims, error)
	KeyID() string
	PublicKey() *ecdsa.PublicKey
}
