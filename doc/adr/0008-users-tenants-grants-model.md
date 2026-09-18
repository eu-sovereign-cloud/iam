# 0008 - Users, Tenants and Grants as first-class entities

## Status

Accepted

## Context

Issue #1 asks for a service that mints JWTs carrying a `tenants` claim (see
ADR 0003). Something has to decide which tenant strings a given subject is
allowed to have stamped into their token — ecp itself has no opinion here,
it has no `Tenant` CRD and simply trusts whatever `tenants` claim it's
given. If IAM only modeled PATs (as originally scoped), tenant membership
would have to be baked into each PAT at creation time, with no way to
revoke a single tenant's access without revoking the whole PAT, and no
separate notion of "who is allowed to manage tenants at all".

## Decision

IAM manages four entities: `User`, `Tenant`, `Grant`, and `PAT`.

- A `User` has a `Subject` (the JWT `sub`) and an `Admin` flag.
- A `Tenant` is just an ID + display name IAM knows about.
- A `Grant` links a `User.Subject` to a `Tenant.TenantID` — it is the
  record of "this subject may claim this tenant". Nothing more.
- A `PAT` belongs to a `User.Subject`. Per ADR 0012 a PAT is itself a signed
  JWT: its `tenants` claim is computed from the subject's current Grants
  once, at issuance time (there is no later exchange step to refresh it —
  a new Grant only affects PATs issued after it).

Authorization within IAM's own API: only Users with `Admin: true` may
create/list/delete Tenants, Users, and Grants. Any User may manage their
own PATs (self-service) regardless of the `Admin` flag — this preserves the
"no interactive login, automation-friendly" goal from issue #1 for ordinary
users, while keeping the tenant/user registry itself under admin control.

**Explicitly deferred**: creating a Grant records IAM's own intent, but does
**not** create the corresponding RBAC `RoleAssignment` object inside ecp's
tenant namespace. Doing so requires IAM to have some access into the ecp
cluster/API and a defined mapping from an IAM grant to an ecp role/scope —
neither exists yet, and no issue tracks it at time of writing. The
`service.GrantService` port carries a `// TODO` marking where that
integration would hook in, so the data model doesn't need to change when it
lands.

## Consequences

- Revoking a Grant only affects *PATs issued after that point* (ADR 0012:
  `tenants` is baked into a PAT at issuance, not resolved per-use); it does
  nothing to any `RoleAssignment` that may already exist in ecp, nor to any
  PAT already issued — until the deferred integration lands, tenant access
  removal is a two-(or three-)system, eventually-consistent operation.
- The admin/self-service split means a compromised non-admin PAT can only
  ever have been issued for its own subject's own current Grants at the
  time it was created; it cannot create Tenants, Users, or Grants for itself.
