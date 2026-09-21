# 0004 - PAT revocation deletes the Secret

## Status

Accepted, but superseded in part by ADR 0012: PATs are now ConfigMaps, not
Secrets (they carry no secret material once there's no token hash to
protect), and there is no more `IAM_ACCESS_TOKEN_TTL`/exchange-time TTL —
revoke-by-delete and the "not found == revoked" logic described below
still stand exactly as written, just against the PAT's own metadata record
directly rather than via a separate short-lived exchanged JWT. See ADR
0012 for the fuller, now-current revocation-gap discussion.

## Context

Issue #1 requires PAT revocation. Two options: delete the PAT's Secret
outright, or keep it and mark it revoked (e.g. a `revoked-at` annotation),
retaining a record for audit purposes.

## Decision

Revoke = delete the Secret. `TokenService.Exchange` and
`PATService.Authenticate` treat "not found" identically to "revoked" — both
simply fail the lookup by hash, so no separate revoked-check branch exists.

To bound the resulting gap — a JWT already minted from a PAT that gets
revoked a moment later remains valid until its own `exp`, since ecp
verifies it offline via signature only — the default access-token TTL is
kept short (`IAM_ACCESS_TOKEN_TTL`, default 15m). Fully closing that gap
requires the gateway to also check a liveness endpoint per request, which
is issue #2's `/userinfo` endpoint, layered in later.

## Consequences

- No audit trail of past PATs beyond whatever external log aggregation
  captures at creation/deletion time; acceptable for a minimal polyfill.
- Simpler code: one code path for "this PAT can't be used", not two.
- The 15-minute default TTL is a meaningful, if partial, mitigation on its
  own even before issue #2 lands — worth keeping short even after `/userinfo`
  exists, since that endpoint is a "liveness/revocation check", not the
  primary verification path.
