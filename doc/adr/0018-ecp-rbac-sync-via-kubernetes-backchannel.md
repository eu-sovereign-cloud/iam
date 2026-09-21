# 0018 - ecp RBAC sync via a Kubernetes backchannel, not ecp's REST API

## Status

Accepted.

## Context

ADR 0008 deferred "creating a Grant should create the corresponding
RBAC `RoleAssignment` in ecp": IAM had no ecp credentials/access, and no
defined IAM-grant → ecp-role mapping. Research into the sibling `ecp`
repo settles both:

- ecp has **no `Tenant` CRD or REST resource**. A tenant is purely a
  string scope; its Kubernetes namespace is `hex(sha3-224(tenantID))`
  (`ecp/framework/backend/kubernetes/adapter.go`'s `ComputeNamespace`),
  auto-provisioned by ecp the first time any tenant-scoped, namespaced
  object (`Role`, `RoleAssignment`, `Workspace`) is written **through
  ecp's own REST API** (`NamespaceManagingWriterAdapter.Create` is
  inline logic in ecp's write path, not a cluster-wide watch/reconcile
  loop — confirmed the hard way, see Decision below). So "IAM creates
  the tenant in ecp" concretely means: **creating a `Role` object in
  that computed namespace** is the ecp-side equivalent of provisioning
  the tenant — but since this backchannel bypasses ecp's REST API
  entirely, IAM has to create that namespace itself first.
- `Role`/`RoleAssignment` are namespaced CRDs, group
  `authorization.v1.secapi.cloud`, version `v1`.
  `RoleSpec{Permissions []{Provider, Resources []string, Verb []string}}`,
  `RoleAssignmentSpec{Subs, Roles []string, Scopes
  []{Tenants, Regions, Workspaces []string}}` — a `RoleAssignment` binds
  subject IDs to role names, scoped by tenant.
- ecp's own Go types for these CRDs
  (`resource/authorization/v1/{role,role-assignment}/backend/kubernetes`)
  are not importable cleanly: `resource/go.mod` requires
  `github.com/eu-sovereign-cloud/ecp/framework v0.0.1`, but no
  `framework/v0.0.1` tag exists anywhere in that repo — only ecp's own
  `go.work`-local `replace` directives make it resolve inside an ecp
  checkout. A `replace` from IAM pointing at a local relative path would
  be fragile (checkout-adjacency-dependent, unpinned) and would pull
  `client-go`/`controller-runtime`/`ecp/framework` transitively into
  IAM's dependency tree for two small structs.
- The namespace hash itself
  (`hex(sha3.New224().Write(tenantID))`) uses Go's **standard library**
  `crypto/sha3` (added in Go 1.24; IAM is on 1.26) — six lines, no new
  dependency.

## Decision

IAM talks to ecp's `Role`/`RoleAssignment` CRDs **directly as Kubernetes
objects, via `k8s.io/client-go/dynamic`**, against the same cluster ecp
runs in — a backchannel, not ecp's REST/HTTP frontend or its external
`go-sdk`. `internal/pkg/kube` gains `BuildDynamicClient`, reusing the
same config resolution as `BuildClientset`; no new go.mod dependency is
needed (`client-go` is already required by `kubestore`).

- New adapter `internal/adapter/kuberbac` (alongside `kubestore`/
  `kubecrypt`): implements the new `ports.TenantRoleStore` port.
  Reimplements the tenant-namespace hash locally (`namespace.go`), and
  defines small local Go structs mirroring the vendored CRD schema at
  `test/e2e/testdata/crds/authorization.v1.secapi.cloud_{roles,role-assignments}.yaml`
  (`gvr.go`) — the same "mirror the wire format, don't import it"
  approach `internal/model/token_scope.go` already uses for ecp's
  `TokenScope`.
- Every tenant gets one IAM-managed `Role` named `model.TenantAdminRole`
  ("tenant-admin", wildcard permissions), created when the tenant is
  created (`CreateTenant`) and deleted when it is (`DeleteTenant`).
- `Grant` gains a required `Roles []string` field: the ecp role names a
  non-tenant-admin subject is bound to within the tenant via a single
  `RoleAssignment` (ecp's `RoleAssignmentSpec.Roles` is itself a list —
  one assignment can already grant several roles at once, so `Grant`
  mirrors that instead of forcing one role per Grant). IAM does not
  create or validate that these roles exist in ecp — they're assumed to
  already exist, created by other means. A tenant-admin `Grant`
  (`Admin: true`, ADR 0016) is instead bound to just
  `[]string{model.TenantAdminRole}`; `SetGrantAdmin` swaps the binding
  between the two on promotion/demotion.
- New `RepairTenant` controller/endpoint
  (`POST /api/v1/tenants/{tenantId}/repair`, same authorization scope as
  `CreateGrant`/`DeleteGrant`): re-applies the canonical `tenant-admin`
  `Role` and every current `Grant`'s `RoleAssignment`, overwriting
  drift, and deletes any IAM-managed `RoleAssignment` (identified by a
  `iam.eu-sovereign-cloud/managed-by` label) that no longer corresponds
  to a `Grant`.
- Every `RoleAssignment` IAM writes sets `Scopes: [{Tenants: [tenantID]}]`
  only — `Regions`/`Workspaces` are left empty. Per ecp's own
  `doc/AUTH.md` ("A `RoleAssignmentScope` covers the request when all
  three dimensions match"), an empty field is a wildcard covering any
  value on that dimension, not "none" — so leaving them empty means "any
  region, any workspace within the tenant," the only correct default
  since IAM has no concept of regions or workspaces at all. Not a gap.
- `DeleteTenant` refuses (`ErrConflict`) while any `Grant` still exists
  for the tenant — the only "is this tenant empty" check IAM can make
  for itself, since it has no visibility into whatever else ecp's
  namespace might hold (`Workspace`s, workloads). It only ever deletes
  the `Role` it manages, never the Kubernetes `Namespace` object itself
  — ecp's own documented namespace auto-cleanup handles that once truly
  empty, in deployments where ecp is actually running.
- `kuberbac.EnsureTenantAdminRole`/`SetRoleAssignment` each call an
  unexported `ensureNamespace` first (get-or-create the tenant's
  namespace as a bare `Namespace` object) before writing into it.
  **Revision, found by a real e2e run against a kind cluster with no
  ecp deployed in it**: the original version of this decision assumed
  writing a `Role`/`RoleAssignment` directly would land in an
  already-provisioned namespace the way it would going through ecp's
  REST API — it doesn't, because that provisioning is inline logic in
  ecp's own write path (see Context), which this backchannel never
  runs. Nothing else creates the namespace, so IAM has to. `DeleteTenant`
  still never deletes the `Namespace` object — only creation needed the
  correction, not the earlier "IAM never touches the Namespace object"
  deletion decision.

## Consequences

- IAM's Kubernetes ServiceAccount needs RBAC permission to read/write
  `roles.authorization.v1.secapi.cloud` and
  `role-assignments.authorization.v1.secapi.cloud` in whatever namespace
  a given tenant hashes to — a wider scope than its current
  ConfigMap/Secret-in-its-own-namespace permissions. Deployment manifests
  need updating alongside this change.
- `internal/adapter/memoryrbac` (mirroring ADR 0017's `memorystore`/
  `memorycrypto`) gives every test a real, in-memory
  `ports.TenantRoleStore` — no fake Kubernetes objects needed for
  controller/service/web tests; `kuberbac`'s own tests use
  `k8s.io/client-go/dynamic/fake`.
- `test/e2e`'s vendored-CRD compatibility check (ADR 0011) is no longer
  purely a schema-compatibility check once IAM itself starts writing
  these objects — a natural follow-up is asserting IAM's actual
  `Role`/`RoleAssignment` output there, not just that a hand-built
  example still applies.
- If ecp's `Role`/`RoleAssignment` schema or its tenant-namespace hashing
  algorithm ever changes, nothing here detects that automatically —
  same accepted maintenance burden ADR 0011 already flags for the
  vendored CRD YAML.
