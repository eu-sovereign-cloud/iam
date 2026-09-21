package controller

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/eu-sovereign-cloud/iam/internal/model"
	"github.com/eu-sovereign-cloud/iam/internal/ports"
)

// GetJWKS publishes IAM's current signing key as a JWKS (issue #2), so
// verifiers can validate IAM-issued JWTs fully offline instead of trusting
// a statically distributed key file.
type GetJWKS struct {
	Signer ports.Signer
}

func (c *GetJWKS) Do(_ context.Context) (model.JWKSet, error) {
	pub := c.Signer.PublicKey()

	// Uncompressed SEC1 point: 0x04 || X || Y, X and Y each fixed-width
	// per curve (32 bytes for P-256) — avoids the deprecated pub.X/pub.Y
	// big.Int accessors.
	raw, err := pub.Bytes()
	if err != nil {
		return model.JWKSet{}, fmt.Errorf("encoding public key: %w", err)
	}
	coord := (len(raw) - 1) / 2
	x, y := raw[1:1+coord], raw[1+coord:]

	return model.JWKSet{Keys: []model.JWK{{
		Kty: "EC",
		Crv: "P-256",
		Use: "sig",
		Alg: "ES256",
		Kid: c.Signer.KeyID(),
		X:   base64.RawURLEncoding.EncodeToString(x),
		Y:   base64.RawURLEncoding.EncodeToString(y),
	}}}, nil
}
