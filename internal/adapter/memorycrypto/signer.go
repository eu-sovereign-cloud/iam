// Package memorycrypto is a non-cryptographic stand-in for ports.Signer,
// for tests that need to exercise PAT issue/verify round-tripping without
// paying for or depending on real ES256 signing. See kubecrypt for the
// production signer.
package memorycrypto

import (
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
