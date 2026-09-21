# 0019 - OIDC discovery + JWKS + `/userinfo`

## Status

Accepted.

## Context

[Issue #2](https://github.com/eu-sovereign-cloud/iam/issues/2) asks IAM
to expose:
- `/.well-known/openid-configuration` + `/.well-known/jwks.json`, so a
  verifier can validate IAM-issued JWTs fully offline instead of trusting
  a manually-distributed key file.
- `/userinfo`, so a verifier that has already checked a JWT's signature
  offline can still catch a PAT revoked since issuance — the one thing
  offline verification alone can't see.

ADR 0005 chose an asymmetric key (ES256) specifically to enable this
later; ADR 0003 explicitly deferred it.

**On the ecp side**: reading the sibling `ecp` checkout, its gateway
currently verifies JWTs against a *statically configured* key file
(`--jwt-secret`), with no OIDC-discovery/JWKS client. Its own
`doc/AUTH-SPEC-REVIEW.md` documents per-request revocation introspection
as a *rejected* design (a network round-trip on the verification hot
path, plus a hard IdP-availability dependency), favoring short-lived
tokens with refresh instead. So nothing consumes these endpoints today —
this is the IAM-side half of a feature ecp would need to separately opt
into (a discovery/JWKS client, and a decision to accept the
introspection trade-off it currently documents rejecting). That doesn't
change what issue #2 asks IAM to build; noted here so a future reader
isn't confused about why nothing currently exercises these routes.

## Decision

- **Top-level, unauthenticated routes, not under `/api/`.**
  `/.well-known/openid-configuration`, `/.well-known/jwks.json`, and
  `/userinfo` are fixed paths by convention (OIDC discovery, RFC 7517,
  and typical OIDC userinfo placement respectively) and must be
  reachable at the service root. `internal/service.Service` gains
  `RegisterDiscoveryRoutes(mux *http.ServeMux)`, called in
  `cmd/iamd/main.go` on the same top-level mux `/api/`- and `/web/`-
  routers are mounted to, rather than living inside `Router()`'s
  `/api/v1/...`-scoped mux. Discovery and JWKS carry no secrets and take
  no auth; `/userinfo` authenticates via the bearer token under test
  itself (`AuthenticatePAT`, the same check `RequireAuth` already does),
  not `RequireAuth`/`AuthenticateUser` — there is no "user" to resolve
  independent of the token being checked.
- **`issuer` == `IAM_JWT_ISSUER`.** The discovery document's `issuer`
  (and the base for `jwks_uri`/`userinfo_endpoint`) reuses the existing
  `IAM_JWT_ISSUER` config value — already minted into every JWT's `iss`
  claim, and per OIDC convention the issuer identifier already has to be
  the service's externally-reachable base URL. No new config field.
- **`ports.Signer` grows `KeyID()`/`PublicKey() *ecdsa.PublicKey`**,
  rather than having each adapter build and expose JWK JSON itself: the
  EC-point-to-JWK transform (RFC 7518 §6.2: base64url of the X/Y
  coordinates from the key's uncompressed SEC1 encoding, `crypto/ecdsa`'s
  non-deprecated `PublicKey.Bytes()`) is generic ECDSA math, not
  adapter-specific, so it lives once in a new `controller.GetJWKS`
  (`Signer ports.Signer`, `Do(ctx) (model.JWKSet, error)`) per ADR 0014's
  controller-holds-logic split. Importing `crypto/ecdsa` into `ports` is
  consistent with the ES256 commitment ADR 0005 already made
  repo-wide, not new coupling.
  `internal/adapter/memorycrypto`'s fake `Signer` (stateless, no real
  key — `Sign`/`Verify` round-trip claims through base64/JSON) gets a
  package-level, generated-once fake ECDSA key purely so `KeyID()`/
  `PublicKey()` have something to return in tests; it carries no
  cryptographic meaning, and `Signer{}` itself stays field-free so its
  zero value stays ready to use.
- **No new controller for `/.well-known/openid-configuration`.** Unlike
  JWKS, the discovery document is pure static-config formatting with
  zero domain logic — built directly in the service handler as a small
  unexported struct, the same way other simple response DTOs already
  are (e.g. `grantResponse` in `internal/service/grants.go`).
- **No new controller for `/userinfo` either.** The existing
  `controller.AuthenticatePAT` (verify signature → look up by `jti` →
  reject if revoked or expired) already does exactly the required check
  and is reused as-is.
- **`/userinfo`'s response is deliberately minimal**: `{"sub": "..."}`
  on success, 401 (with `WWW-Authenticate: Bearer error="invalid_token"`,
  RFC 6750) on failure. The issue frames this explicitly as a
  liveness/revocation check, not the primary verification path — a
  caller has already gotten the full claims (tenants, scope, ...) from
  offline JWKS verification, so `/userinfo` doesn't need to repeat them.
- **`GET` only** for all three routes. OIDC userinfo endpoints commonly
  also accept `POST`; not adding it here narrows the surface to what's
  actually asked for, and it's a backward-compatible addition later if
  ever needed.
- **`Cache-Control: max-age=300`** on the JWKS response — the key
  rotates rarely (ADR 0005 has no rotation story yet), so a short cache
  cheaply spares verifiers a round trip on every request.

## Consequences

- IAM's public HTTP surface now includes three genuinely unauthenticated
  routes (discovery + JWKS by design; `/userinfo` self-authenticates via
  the token it's checking) — worth keeping in mind for anything that
  assumes every route sits behind `RequireAuth`.
- As noted in Context, nothing calls these yet from ecp's side; that's a
  separate, not-yet-scoped follow-up on the ecp side, not a gap in this
  work.
- If IAM's signing key is ever rotated in place (still out of scope —
  ADR 0005's Secret holds exactly one key pair), `GetJWKS` as written
  only ever publishes the current key; a rotation story would need to
  publish the previous key too for some overlap window, which this ADR
  does not attempt to solve.
