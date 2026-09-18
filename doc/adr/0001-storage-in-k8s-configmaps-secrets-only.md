# 0001 - Storage in Kubernetes ConfigMaps and Secrets only

## Status

Accepted

## Context

IAM is a deployment-time polyfill for IONOS's self-installable model, not a
general-purpose product (see the issue #1 "out of scope" note). It runs
alongside ecp in the same cluster. ecp itself persists its domain resources
as CRDs backed by a full controller-runtime reconciliation stack, but that
machinery exists to manage cloud resources with complex lifecycles
(networks, workloads, RBAC). IAM's own state — Users, Tenants, Grants, PATs,
one signing key — is much simpler: flat records with no spec/status split
and no external effects to reconcile.

## Decision

IAM stores everything as native Kubernetes ConfigMaps (non-sensitive
metadata: Users, Tenants, Grants) and Secrets (PAT hashes, the JWT signing
key) in a single namespace, using the core `client-go` typed clientset
directly. No CRDs, no controller-runtime, no external database. See the
design plan's "Data model & Kubernetes storage" section for the exact
resource shapes.

## Consequences

- Zero extra infrastructure to run IAM: any cluster with an API server is
  enough, no database, no CRD installation step.
- No schema validation, defaulting, or admission webhooks the way a CRD
  would give for free; validation is IAM's own responsibility in the
  service layer.
- Trades away CRD ergonomics (`kubectl get iamusers`, etc.) for simplicity;
  acceptable since this is IAM's own private state, not something operators
  are expected to interact with via kubectl.
