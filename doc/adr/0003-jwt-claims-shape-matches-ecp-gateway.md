# 0003 - JWT claims shape matches the ecp gateway exactly

## Status

Accepted

## Context

IAM's only reason to exist is to mint JWTs the SECA gateway in `ecp` will
accept in place of a CSP-issued one. The gateway's actual, enforced
expectation lives in `ecp/gateway/internal/authn/jwtstd.go`, not in the SECA
spec (the spec deliberately leaves the claim schema open). That code:

- Requires standard `jwt.RegisteredClaims`: `sub` (required, else 401),
  `exp` (always required), plus `iss`/`aud`, which are enforced (and made
  mandatory) only when the gateway operator configured an expected value.
- Reads an optional `tenants` claim (`[]string`) as issuer-asserted tenant
  membership.
- Reads an optional `scope` claim shaped like `resource.TokenScope`
  (`{"tenants":[...],"regions":[...],"workspaces":[...]}`, all optional) as
  a down-scoping cap. A plain OAuth2 string `scope` value is tolerated and
  ignored, never rejected.
- Does **not** implement OIDC discovery/JWKS; the verification key today is
  a static file. Dynamic key discovery is issue #2, layered on top of
  whatever this service issues.

ecp has no `Tenant` CRD or registry of its own — a tenant is just a string
it trusts from the `tenants` claim and hashes into a namespace name. That
tells IAM what shape to *emit*, not how to *decide* what a subject is
allowed to have stamped into that claim; IAM still needs its own notion of
which tenants a subject may claim (see ADR 0008).

## Decision

`internal/model.Claims` embeds `jwt.RegisteredClaims` and adds `Scope
*model.TokenScope` and `Tenants []string`, with identical JSON tags to
ecp's `jwtClaims`/`tokenScopeClaim`. `model.TokenScope` is a local
reimplementation of ecp's `resource.TokenScope` (same three fields, same
tags) rather than an imported dependency — three fields don't justify a
cross-repo Go module dependency, and it decouples IAM's release cadence
from ecp's.

At issuance: `sub` = the User's subject, `iss`/`aud` = from IAM's config,
`exp` = now + configured access-token TTL, `tenants` = the subject's
current Grants (resolved fresh at exchange time, not cached on the PAT —
see the PAT/Grant relationship in ADR 0008), `scope` = the PAT's own
optional down-scope cap, copied through verbatim.

## Consequences

- Any change to ecp's claims contract must be mirrored here by hand; there
  is no shared Go type to keep the two in sync automatically. Acceptable
  given how small and stable this contract is expected to be.
- Because tenant membership is resolved at exchange time rather than baked
  into the PAT, revoking a Grant takes effect on the *next* token exchange
  without needing to touch or revoke the PAT itself.
