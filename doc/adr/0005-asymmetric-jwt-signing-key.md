# 0005 - Asymmetric (ES256) JWT signing key

## Status

Accepted

## Context

IAM must sign JWTs the gateway can verify. HMAC (a shared secret) is
simpler operationally for a two-party system, but issue #2 will need IAM to
expose a JWKS endpoint publishing its public signing key(s) so the gateway
can verify offline without an HMAC secret ever leaving IAM.

## Decision

Use ES256 (ECDSA P-256), matching one of the signing methods
`ecp/gateway/internal/authn/jwtstd.go` already supports via
`jwt.GetSigningMethod`/`ParseVerifyKey`. The key pair is generated on first
startup if the `iam-signing-key` Secret doesn't exist yet, storing both the
PEM-encoded private and public key plus a random `kid`, and is loaded from
that same Secret on every subsequent startup (`kubecrypt.LoadOrCreate`).

## Consequences

- A future JWKS endpoint can read the public key straight out of the same
  Secret this ADR creates, with no re-keying or migration needed.
- Losing the `iam-signing-key` Secret invalidates every previously issued
  JWT's ability to be verified against a freshly generated key — there is
  no key backup/rotation story yet; acceptable for a first version, but
  worth revisiting once this runs in a real deployment.
- A startup race between two instances creating the Secret simultaneously
  is handled by falling back to reading whichever one won
  (`kubecrypt.LoadOrCreate`), though ADR 0009 means only one instance
  should be running at a time regardless.
