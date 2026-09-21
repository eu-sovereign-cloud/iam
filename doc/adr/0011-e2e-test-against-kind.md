# 0011 - End-to-end test against a real cluster (kind), in CI

## Status

Accepted

Note: the "IAM itself still does not create `RoleAssignment` objects"
statement in the Decision section below no longer holds — ADR 0018 has
IAM create/manage real `Role`/`RoleAssignment` objects directly. The
vendored-CRD compatibility check described here (`applyDemoRoleAssignment`)
still exists as-is and is still useful (it predates and is independent of
IAM's own writes), but asserting IAM's *own* `Role`/`RoleAssignment`
output here too is a natural follow-up, not yet done.

## Context

`internal/controller/controller_test.go` already exercises the full HTTP
flow (create tenant → grant → self-service PAT issuance → revoke; see ADR
0012 — a PAT is the JWT directly, there is no separate exchange step)
against a fake Kubernetes clientset (`k8s.io/client-go/kubernetes/fake`). That's a
fast, hermetic test of IAM's own business logic, but it can't catch bugs in
how IAM talks to a *real* API server — which is exactly how a real bug was
found: `adapter.BuildClientset`'s out-of-cluster kubeconfig fallback was
broken (`clientcmd.BuildConfigFromFlags("", "")` does not search
`~/.kube/config`/`KUBECONFIG` the way `kubectl` itself does), and the fake
clientset in the unit test naturally can't exercise that code path since
it's never asked to resolve a real kubeconfig at all.

Manually running `iamd` against a real cluster and driving it with `curl`
caught that bug. This ADR captures making that manual walkthrough into an
automated, repeatable test.

## Decision

- A build-tagged Go test package, `test/e2e` (`//go:build e2e`), so
  `go test ./...` never picks it up by accident. `TestEndToEnd` builds the
  real `iamd` binary, starts it as a subprocess against whatever cluster
  the ambient kubeconfig points to, and drives it purely over HTTP with
  `testify` assertions — the same interface a real operator or the ecp
  gateway would use. It also restarts the process mid-test to verify ADR
  0009's load-at-startup cache actually survives a restart.
- Uses `testify` and plain `net/http`/`encoding/json`, not a generated API
  client or a k8s Go clientset of its own — cluster-side setup/teardown
  (installing/removing the two vendored CRDs, deleting the test namespace)
  shells out to the `kubectl` binary, consistent with the project's
  low-tech bias (see the original architecture discussion) and avoiding an
  extra k8s client dependency in test code that never otherwise needs one.
- CI runs this in kind (`.github/workflows/e2e.yml`, `helm/kind-action`),
  as its **own workflow, separate from `ci.yml`, and `continue-on-error:
  true`** — mirroring ecp's own `conformance.yaml`, which is likewise a
  separate, non-blocking workflow. A kind cluster spinning up is more prone
  to CI infrastructure flakiness than a plain `go build`/`go vet`/`go test`
  gate; a flaky cluster shouldn't block every PR the way a real lint/test
  regression should.
- `make e2e` runs the same test locally against whatever cluster the
  caller's kubeconfig points to (kind, a real cluster, anything) — the
  Makefile target and the CI step call the identical command, per the
  project's existing convention of Makefile-as-source-of-truth for what a
  check means.
- The two ecp CRDs (`Role`, `RoleAssignment`) are vendored verbatim into
  `test/e2e/testdata/crds/` from `eu-sovereign-cloud/ecp`'s
  `charts/ecp/crds/` (see that directory's own `README.md` for the exact
  provenance and how to refresh them). That directory is genuinely what ecp
  itself uses too — it's controller-gen's output, and the same directory
  ecp's own `envtest`-based tests point at — not a hand-authored
  substitute. The test applies a `RoleAssignment` shaped exactly like ADR
  0008's deferred "Grant → RoleAssignment" integration point would produce,
  purely to confirm the vendored schema still accepts it. **IAM itself
  still does not create `RoleAssignment` objects** — this is a compatibility
  check on the vendored copy, not new IAM behavior.

## Consequences

- Catches an entire class of bug the fake-clientset test structurally
  cannot: anything about how IAM talks to a *real* API server (kubeconfig
  resolution, actual object shapes accepted by a real apiserver, real
  namespace lifecycle).
- Vendoring two YAML files from another repo creates a small, explicit
  maintenance burden: if ecp's `Role`/`RoleAssignment` schema changes,
  nothing here detects that automatically — only a subsequent e2e test run
  failing to apply the example `RoleAssignment` would surface it. Accepted
  because these CRDs are simple, roughly stable, and the check they enable
  (vendored-copy-still-works) is valuable enough to outweigh manually
  refreshing two files occasionally.
- Slower and flakier than the unit/integration test suite by nature (real
  cluster bring-up, real process start/stop) — this is exactly why it runs
  as a separate, non-blocking workflow rather than folded into `ci.yml`.
- If IAM ever depends on more of ecp's behavior (not just CRD schema
  compatibility), revisit whether `test/e2e` should grow into something
  closer to ecp's own conformance suite rather than this single test.
