# 0012 - PATs are the issued JWTs directly (no exchange step)

## Status

Accepted. Supersedes the exchange-based approach described in ADR 0003 and
ADR 0004 (both left in place, with pointers to this ADR, for the historical
reasoning — this document is the current source of truth for how PATs and
JWTs relate).

## Context

The original design (ADR 0003/0004) split the credential into two tiers: a
long-lived opaque PAT (`iampat_...`, a random secret whose hash was stored)
that got *exchanged*, via `POST /api/v1/tokens`, for a short-lived signed
JWT. Revoking the PAT immediately blocked new exchanges, bounding a leak's
blast radius to at most one JWT lifetime (15 minutes by default).

That design was reconsidered: the PAT should *be* the JWT directly — you
create one and that string is your bearer credential everywhere, both
against IAM's own API/web UI and against `ecp`'s gateway, with no separate
exchange call.

This has a real, deliberately accepted cost. `ecp`'s gateway verifies JWTs
**offline** — signature and `exp` only, no callback to IAM
(`ecp/gateway/internal/authn/jwtstd.go`). Under the two-tier design,
revoking the long-lived PAT stopped new short-lived JWTs from being minted,
which is what actually bounded a leak. Once the PAT *is* the JWT, that
mechanism disappears: revoking a PAT in IAM can only stop it from
authenticating **against IAM itself**; a copy of the same JWT already held
by `ecp` (or anything else) keeps working until its own `exp`, no matter
what IAM does, until issue #2's `/userinfo` liveness check exists to give
the gateway a way to ask.

## Decision

- **`PATService.Create` builds and signs the JWT directly**: `sub`=subject,
  `iss`/`aud` from config, `iat`/`nbf`=now, `jti`=a new ID, `tenants`=the
  subject's Grants *resolved once, at this moment*, `scope`=the optional
  down-scope cap. That signed string is returned once, as the PAT's
  "secret" — same UX as before, different content, and never persisted.
- **Tenant claims are frozen at issuance, not resolved per-use.** There is
  no later "exchange" moment to refresh them anymore. Granting or revoking
  a Grant after a PAT is issued does **not** change that PAT's embedded
  `tenants` claim — the user must issue a new PAT to pick up new grants.
  This is stated plainly in the web UI's issue-token panel copy, not left
  implicit.
- **No forced expiry cap.** Leaving the `ttl` field blank mints a 100-year
  expiry rather than a short mandatory one — a JWT's `exp` can never be
  omitted (`ecp` requires it), so "no expiry" means "effectively never," not
  "absent." This matches issue #1's explicit goal of long-lived,
  non-interactive credentials for CI/automation, at the cost described below.
- **PATs move from Secrets to ConfigMaps** (amending ADR 0001's example):
  the only thing that made a PAT record secret-shaped was its token hash,
  which no longer exists — IAM never persists the raw JWT or a hash of it,
  only `id` (the `jti`), `subject`, `name`, `created-at`, `expires-at`, and
  an optional `scope`.
- **`Signer` gains `Verify`** (parse + validate ES256 signature + mandatory
  expiry, mirroring `ecp`'s own verification style). `PATService.Authenticate`
  now verifies the JWT, then checks its `jti` is still present in the PAT
  store — this *is* an effective, immediate revocation check, it just can't
  reach past IAM's own boundary. This is exactly the shape issue #2's public
  `/userinfo` endpoint will eventually expose to `ecp` itself, so this
  change sets that work up rather than working against it.
- **`TokenService`, `POST /api/v1/tokens`, and `adapter.TokenGenerator` are
  deleted** — there is no exchange operation left to perform.
- **Bootstrap admin issuance moves from `internal/adapter` to
  `internal/service`** (`service.EnsureBootstrapAdmin`): minting a PAT is
  now real claims-building-and-signing business logic that orchestrates
  `UserService` and `PATService`, not something the adapter layer should
  hand-roll — a layering correction made in passing while this code was
  already being rewritten.

## Consequences

- **The core trade-off, accepted explicitly**: a leaked PAT is valid for
  its entire signed lifetime against anything that verifies it offline
  (i.e. `ecp`, today), regardless of what's revoked in IAM. IAM's own
  revocation is real and immediate, but only for IAM's own API/web auth.
  This gap closes once issue #2's `/userinfo` exists and `ecp` is wired to
  check it; until then, operators should treat a PAT leak as live until
  expiry from `ecp`'s perspective, not just until "revoked" in IAM.
- Granting a user new tenant access requires them to issue a new PAT to use
  it — an existing long-lived PAT does not "pick up" new grants on its own.
- PAT storage no longer needs the hash-generation/comparison machinery
  (`adapter.TokenGenerator`) at all — one less moving part.
- `IAM_ACCESS_TOKEN_TTL` (the old fixed exchange-time JWT lifetime) is
  removed from config entirely; TTL is now purely a per-PAT, per-creation
  choice (or the 100-year default).
