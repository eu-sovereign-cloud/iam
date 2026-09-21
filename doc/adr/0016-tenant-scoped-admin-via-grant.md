# 0016 - Tenant-scoped admin as a flag on Grant

## Status

Accepted.

## Context

ADR 0008 gives IAM a single privilege: `User.Admin`, global and all-or-
nothing. A user is either trusted to manage every Tenant/User/Grant, or
trusted with nothing beyond their own PATs. That's too coarse once a
deployment has multiple tenants managed by different people — a customer
who owns one tenant has no way to manage membership of *their* tenant
without being handed the keys to every other tenant too.

## Decision

Tenant admin is a boolean on `Grant`, not a new field on `User`:

```go
type Grant struct {
	Subject   string
	TenantID  string
	GrantedAt time.Time
	GrantedBy string
	Admin     bool
}
```

`Grant` already is the Subject↔TenantID link (ADR 0008); tenant-admin
status is inherently scoped to one tenant, so it belongs on the record
that already expresses "this subject relates to this tenant" rather than
as a list-like field bolted onto `User`. It also means `model.User` and
the identity `model.WithIdentity` stashes in `ctx` at auth time don't
change shape — the tradeoff is that checking tenant-admin status needs a
`GrantStore` lookup, unlike the pure-`ctx` `model.RequireAdmin`/
`RequireSelfOrAdmin`. Since `internal/model` must not depend on
`internal/ports` (ADR 0002), that check lives as an unexported
`requireTenantAdmin(ctx, grants ports.GrantStore, tenantID string) error`
helper in `internal/controller` instead, alongside the two pure `model`
checks it complements.

Authorization changes:

- `CreateGrant`/`DeleteGrant`: global admin, **or** a subject holding an
  admin `Grant` for that specific `tenantID`, may create/delete grants
  for that tenant. A tenant admin has no reach into any other tenant.
- New `SetGrantAdmin` controller (mirrors `SetUserAdmin`'s get→mutate→
  update shape) toggles `Grant.Admin`. **Global-admin-only** — a tenant
  admin cannot promote or demote tenant-admin status, even within their
  own tenant, to avoid a privilege-escalation loop.
- `CreateTenant`/`ListTenants`/`DeleteTenant` and every `User`-entity
  controller stay global-admin-only: creating/deleting the Tenant/User
  entities themselves, and global user management, are out of scope for
  a tenant admin.
- `ListUserGrants` is unaffected — it's scoped by `subject`
  (self-or-admin), not by tenant.

Exposed as `PATCH /api/v1/users/{subject}/grants/{tenantId}` (JSON API,
mirroring the existing `PATCH /api/v1/users/{subject}` admin toggle) and
a button in the web UI's user-detail grants table, visible only to
viewers who are themselves a global admin.

ecp-side RBAC integration remains deferred, per ADR 0008's existing note
— a tenant admin's privilege is scoped to IAM's own API only.

## Consequences

- A compromised tenant-admin PAT can grant/revoke access to its one
  tenant, but cannot touch any other tenant, cannot create/delete
  Tenants or Users, and cannot mint further tenant admins.
- Checking `CreateGrant`/`DeleteGrant`'s authorization now costs a
  `GetGrant` lookup for non-global-admin callers (a single in-memory map
  read in `kubestore`, per ADR 0009's load-at-startup cache — no added
  Kubernetes API latency).
