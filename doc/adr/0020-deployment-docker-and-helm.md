# 0020 - Deployment: Docker image + Helm chart

## Status

Accepted.

## Context

iam had a `Dockerfile` (distroless `scratch`, nonroot UID 65532,
statically built) and a CI workflow (`.github/workflows/image-release.yml`)
that already build and push `ghcr.io/eu-sovereign-cloud/iam:<tag>` on a
`v*` tag, but no supported way to actually install it into a cluster: no
Helm chart, no RBAC manifests, no Service, no health checks. This ADR
covers what was added to close that gap.

**Workload shape.** Reading `internal/adapter/kubestore` and
`internal/adapter/kubecrypt` confirms iamd holds no state on local disk
at all: both persist exclusively through the Kubernetes API in
`IAM_NAMESPACE` (state as ConfigMaps, the ES256 signing key as a Secret
named `iam-signing-key`, per ADR 0001/0005). A `StatefulSet`'s defining
features — stable per-pod network identity, per-replica
PersistentVolumeClaims — buy nothing here. What does matter, per ADR
0009, is that iam loads all state once at startup with no watch/
informer: running more than one replica would mean divergent in-memory
caches, and two pods could race creating `iam-signing-key` on a cold
start.

**RBAC scope.** Reading `internal/adapter/kuberbac` (ADR 0018) shows two
genuinely different scopes are needed: `configmaps`/`secrets` are
confined to `IAM_NAMESPACE`, but `kuberbac.ensureNamespace` and the
`Role`/`RoleAssignment` CRD writes target *arbitrary tenant namespaces*
(`hex(sha3-224(tenantID))`), and the `namespaces` resource itself is
cluster-scoped by nature — no single namespaced `Role` can grant that.

**Health checks.** None existed; `cmd/iamd/main.go` only ever registered
`/api/`, `/web/`, and the OIDC discovery routes (ADR 0019).

## Decision

- **A `Deployment`, not a `StatefulSet`**, with `replicas: 1` fixed (not
  yet exposed as a scaling knob) and `strategy: Recreate` (no leader
  election exists, so avoid two pods briefly overlapping during a
  rolling update). Multi-replica support is out of scope for this ADR —
  it would need ADR 0009 revisited (a watch/informer, or leader
  election) first, not just a chart change.
- **`GET /healthz`**, added directly in `cmd/iamd/main.go` (no new
  package — a two-line inline handler like the file's existing routes),
  registered only after `kubestore.Load` and `kubecrypt.LoadOrCreate`
  have both already succeeded, same as every other route. It's
  unauthenticated and used as both the liveness and readiness probe
  target — iamd has no "ready but not live" state distinct from "up at
  all" to justify two different checks.
- **Two RBAC objects, not one**: a namespace-scoped `Role`/`RoleBinding`
  (`configmaps`, `secrets`, and `namespaces` get/create, all evaluated
  against `IAM_NAMESPACE`) plus a cluster-scoped `ClusterRole`/
  `ClusterRoleBinding` (`namespaces` get/create again — needed
  cluster-wide since tenant namespaces are computed, not fixed — and
  `roles`/`role-assignments.authorization.v1.secapi.cloud`). The
  `ClusterRole`/`ClusterRoleBinding` names are suffixed with the release
  namespace (`{{ iam.fullname }}-{{ .Release.Namespace }}`) since
  cluster-scoped objects aren't namespaced and multiple installs need
  distinct names.
- **Helm chart at `deploy/helm/iam/`**, `apiVersion: v2`. `values.yaml`
  maps 1:1 onto `internal/config.Config`: `config.jwtIssuer` has no
  default and is enforced via Helm's `required` template function,
  mirroring `IAM_JWT_ISSUER,required`; `config.namespace` defaults to
  the release namespace; `image.tag` defaults to `.Chart.AppVersion`.
  `KUBECONFIG` is deliberately not templated — in-cluster, `client-go`
  auto-discovers the pod's ServiceAccount token (`kube.BuildClientset`
  already supports this; it's only used for out-of-cluster local dev).
- **Chart and image release together**: `.github/workflows/image-release.yml`
  (same `v*` tag trigger) gained `helm package`/`helm push` steps
  pushing to `oci://ghcr.io/eu-sovereign-cloud/charts`, reusing the same
  `docker/login-action`-established ghcr.io credentials rather than a
  second, independently-versioned release pipeline — one tag means one
  version of both artifacts.

## Consequences

- No HA story yet — `replicaCount` is fixed at 1 and not meant to be
  overridden until ADR 0009's single-replica assumption is itself
  revisited; the chart flags this in a comment rather than silently
  allowing a broken multi-replica install.
- The `ClusterRole`/`ClusterRoleBinding` this chart creates grants iam's
  ServiceAccount write access to `Role`/`RoleAssignment` objects in
  *any* namespace in the cluster (not just ones it manages) — an
  accepted consequence of tenant namespaces being computed rather than
  enumerable, same trust boundary ADR 0018's backchannel already
  established; worth keeping in mind for any future multi-tenant
  cluster-sharing scenario.
- `helm uninstall` never deletes the `iam-signing-key`/state Secrets'
  namespace or any tenant namespaces `kuberbac` created — consistent
  with ADR 0018's `DeleteTenant` already never deleting the `Namespace`
  object itself.
