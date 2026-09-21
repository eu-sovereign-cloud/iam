// Package memorycrypto is a non-cryptographic stand-in for ports.Signer,
// for tests that need to exercise PAT issue/verify round-tripping without
// paying for or depending on real ES256 signing. See kubecrypt for the
// production signer.
package memorycrypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// Signer round-trips model.Claims through JSON + base64 instead of real
// signing/verification — enough to exercise CreatePAT/AuthenticatePAT's
// orchestration logic (sign at creation, verify at authentication)
// without needing real crypto in tests. Stateless; the zero value is
// ready to use.
type Signer struct{}

// fakeKey backs KeyID/PublicKey only — Sign/Verify never use it, since
// this Signer doesn't do real cryptography. It exists purely so tests
// exercising the JWKS endpoint (issue #2) have a public key to publish;
// it carries no cryptographic meaning. Generated once at package init so
// Signer itself can stay field-free (the zero value stays ready to use).
var fakeKey = must(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))

const fakeKeyID = "memory-fake-key"

func must(key *ecdsa.PrivateKey, err error) *ecdsa.PrivateKey {
	if err != nil {
		panic(err)
	}
	return key
}

func (Signer) Sign(claims model.Claims) (string, error) {
	raw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	return "fake." + base64.RawURLEncoding.EncodeToString(raw) + ".signed", nil
}

func (Signer) Verify(token string) (model.Claims, error) {
	var claims model.Claims
	body := strings.TrimSuffix(strings.TrimPrefix(token, "fake."), ".signed")
	if body == token {
		return model.Claims{}, fmt.Errorf("not a fake-signed token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return model.Claims{}, err
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return model.Claims{}, err
	}
	// Expiry is deliberately not checked here: AuthenticatePAT's own
	// pat.Expired(clock.Now()) check (against the caller-controlled
	// Clock) is what callers rely on for expiry behavior, not wall-clock
	// time.
	return claims, nil
}

// KeyID returns a fixed, arbitrary kid — see fakeKey.
func (Signer) KeyID() string { return fakeKeyID }

// PublicKey returns a fixed, arbitrary public key — see fakeKey.
func (Signer) PublicKey() *ecdsa.PublicKey { return &fakeKey.PublicKey }
