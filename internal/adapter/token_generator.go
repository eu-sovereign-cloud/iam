package adapter

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// patPrefix marks IAM-issued PATs, letting them be told apart at a glance
// from other bearer secrets (e.g. in logs) without decoding anything.
const patPrefix = "iampat_"

// TokenGenerator generates random PAT secrets and hashes them with SHA-256.
// The raw secret is only ever returned once, at creation time; only its
// hash is persisted (ADR 0001's storage constraint plus ordinary credential
// hygiene).
type TokenGenerator struct{}

func NewTokenGenerator() TokenGenerator { return TokenGenerator{} }

func (TokenGenerator) NewToken() (raw string, hash string) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("adapter: reading random bytes: " + err.Error())
	}
	raw = patPrefix + base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashToken(raw)
}

func (TokenGenerator) Hash(raw string) string {
	return hashToken(raw)
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
