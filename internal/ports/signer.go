package ports

import "github.com/eu-sovereign-cloud/iam/internal/model"

// Signer mints and verifies the signed JWTs that PATs are (ADR 0012).
type Signer interface {
	Sign(claims model.Claims) (string, error)
	Verify(token string) (model.Claims, error)
}
