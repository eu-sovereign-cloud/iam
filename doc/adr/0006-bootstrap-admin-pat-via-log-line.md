# 0006 - Bootstrap admin PAT delivered via a one-time log line

## Status

Accepted

## Context

Creating a User or Tenant requires an admin PAT, but the very first admin
PAT has no admin PAT to be created by — a bootstrapping problem. Two
reasonable options: log the generated credential once on first startup, or
write it into a separate, clearly-named bootstrap Secret for the operator
to fetch and delete.

## Decision

On startup, `adapter.EnsureBootstrapAdmin` checks whether any User has
`Admin: true`. If none exists, it creates one (`subject: "admin"`) and a
PAT for it, and `cmd/iamd` logs the raw PAT once at `WARN` level. Only the
PAT's hash is ever persisted (same as any other PAT, ADR 0001).

## Consequences

- The operator must have access to iamd's startup logs to retrieve the
  credential; if missed, recovery requires deleting the bootstrap admin
  User's ConfigMap and PAT Secret by hand and restarting iamd to
  regenerate them (or, once a second admin exists, creating a User/PAT the
  normal way).
- Simpler than a dedicated bootstrap Secret: no extra resource type, no
  "did the operator remember to delete it" cleanup step. Log retention is
  the deployer's existing concern, not a new one IAM introduces.
