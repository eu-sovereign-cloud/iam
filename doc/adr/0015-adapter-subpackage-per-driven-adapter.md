# 0015 - One subpackage per driven adapter under internal/adapter

## Status

Accepted.

Note: `kube.go` and `k8shelpers.go`, described below as staying directly
under `internal/adapter`, later moved again to `internal/pkg/kube`
(package `kube`) — they're generic Kubernetes client-go plumbing with no
IAM domain knowledge, not an adapter themselves, so they don't belong
under `internal/adapter` at all. `internal/adapter` now holds only
subpackages, no files of its own. Everything else below — the
kubestore/kubecrypt/system split and the naming convention — still
stands.

## Context

`internal/adapter` held every driven adapter's code as one flat package:
the Kubernetes ConfigMap/Secret-backed `Store` (ADR 0001, ADR 0009), the
ES256 JWT `Signer` (ADR 0005, ADR 0012), and `SystemClock`, plus the
`k8shelpers.go`/`kube.go` plumbing all of them leaned on. That was fine
while there was exactly one adapter per port. It stops scaling the moment
a port gets a second implementation (a non-Kubernetes store for local
dev, a different signing backend, etc.) — everything would still share
one package name and one set of files to search through.

## Decision

- `internal/adapter/kubestore`: the ConfigMap/Secret-backed `Store` and
  everything only it needs — `store.go`, `tenants.go`, `users.go`,
  `grants.go`, `pats.go`, `names.go`, `labels.go`, `store_test.go`, and its
  own `assertions.go` (`ports.UserStore`/`TenantStore`/`GrantStore`/
  `PATStore` satisfaction checks). Constructor renamed `NewStore` →
  `New`, since the package name already says "store" —
  `kubestore.New(...)` reads the same as `kubestore.NewStore(...)` did,
  without repeating itself.
- `internal/adapter/kubecrypt`: the ES256 JWT `Signer`/`Verify` pair —
  `signer.go` and its own `assertions.go` (`ports.Signer` check).
  Constructor renamed `LoadOrCreateSigner` → `LoadOrCreate` for the same
  reason (`kubecrypt.LoadOrCreate(...)`).
- `internal/adapter/system`: `SystemClock` → `Clock` (dropping the
  now-redundant "System" prefix — `system.Clock` says the same thing),
  plus its own `assertions.go` (`ports.Clock` check).
- `internal/adapter` itself keeps only what's genuinely shared across
  adapter subpackages: `kube.go` (`BuildClientset`, used once in
  `cmd/iamd/main.go` to build the clientset passed into both `kubestore`
  and `kubecrypt`) and `k8shelpers.go`, now exporting its small
  `metav1`/`apierrors` wrappers (`MetaGetOpts`, `IsNotFound`, etc.) so
  `kubestore` and `kubecrypt` can both call them without duplicating
  seven one-line functions. Its own `assertions.go` (the old
  `Signer`/`Clock` checks) is deleted — nothing left at this level to
  assert.
- Naming convention going forward: a subpackage's main constructor drops
  any word already said by the package name (`kubestore.New`, not
  `kubestore.NewStore`; `kubecrypt.LoadOrCreate`, not
  `kubecrypt.LoadOrCreateSigner`), and a type named after its role rather
  than repeating the package (`system.Clock`, not `system.SystemClock`) —
  callers read `package.Identifier` together, so the identifier shouldn't
  restate what the package name already established.

## Consequences

- Adding a second implementation of any port (an in-memory store for
  local dev, a different signer) is "add a sibling subpackage," not
  "figure out how to name two different `Store`/`Signer` types sharing
  one package."
- `cmd/iamd/main.go` and every test constructing a full stack
  (`internal/service/service_test.go`, `internal/web/web_test.go`) now
  import three adapter subpackages instead of one `adapter` package;
  a small increase in import-list length in exchange for each adapter's
  code living somewhere its name already tells you.
- ADR 0013's file paths for `SystemClock`/`assertions.go` describe where
  those things lived at the time; this ADR is what moved them again.
