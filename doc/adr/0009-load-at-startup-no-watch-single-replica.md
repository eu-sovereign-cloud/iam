# 0009 - Load state at startup, no watch/informer, single replica

## Status

Accepted

## Context

IAM's data volume (Users, Tenants, Grants, PATs for a self-install
deployment) is small, and its read path (every API request's bearer-auth
check, every admin page load) is latency-sensitive relative to the
Kubernetes API. Building a
full watch/informer/reconciliation loop (the way ecp's `GenericController`
does for CRDs) is disproportionate machinery for four flat resource kinds
with no external side effects to reconcile.

## Decision

`adapter.Store.Load` performs one LIST per resource kind at startup and
keeps the result in an in-memory cache (`sync.RWMutex`-guarded maps). All
reads are served from that cache. Every mutating method
(Create/Update/Delete) writes to the Kubernetes API first, and only updates
the cache once that write succeeds — so the API is always the source of
truth for a successful write, and the cache is always consistent with what
IAM itself has done, though not with what anything else might do to those
objects. There is no watch, no resync, no reconcile loop.

## Consequences

- **Only a single iamd replica may run at a time.** A second replica would
  load its own independent cache at its own startup time and the two would
  silently diverge as each processes writes the other doesn't see. Nothing
  in this design detects or prevents running two replicas; it is an
  operational constraint (e.g. a Deployment with `replicas: 1` and
  `strategy: Recreate`), not one iamd enforces itself.
- **External edits require a restart.** If someone edits or deletes one of
  IAM's ConfigMaps/Secrets directly (`kubectl edit`, `kubectl delete`)
  while iamd is running, the in-memory cache won't reflect it until iamd
  restarts and reloads. IAM's own API/web UI are the intended way to
  mutate this state; direct cluster access is an escape hatch, not a
  supported path.
- Startup cost scales with the number of Users/Tenants/Grants/PATs (one
  LIST per kind); acceptable for the expected scale of a self-install
  deployment's user base.
- Much simpler adapter code: no resync period, no event queue, no
  leader-election story to keep two replicas' caches from diverging.
