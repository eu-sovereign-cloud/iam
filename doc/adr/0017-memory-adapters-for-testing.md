# 0017 - memorystore/memorycrypto: in-memory adapters for tests

## Status

Accepted.

## Context

Before this, tests needing an implementation of the store/signer ports
either hand-rolled their own throwaway doubles (`internal/controller`'s
`fakeUserStore`/`fakeTenantStore`/`fakeGrantStore`/`fakePATStore`/
`fakeSigner` in `fakes_test.go`) or stood up the real
`kubestore.Store`/`kubecrypt.Signer` against a fake Kubernetes clientset
(`internal/service`'s and `internal/web`'s route-level tests) purely
because that was the only other thing available — the Kubernetes layer
there is incidental plumbing, not something those tests care about.

## Decision

Two new adapters live under `internal/adapter`, alongside `kubestore`/
`kubecrypt`/`system` (same one-subpackage-per-adapter shape, ADR 0015):

- `internal/adapter/memorystore`: a single `Store` type implementing
  every store port (`UserStore`, `TenantStore`, `GrantStore`, `PATStore`)
  on plain `map`s guarded by one `sync.Mutex` — no persistence, no
  Kubernetes. It's the promoted, shared version of what
  `internal/controller`'s fakes already did.
- `internal/adapter/memorycrypto`: a stateless `Signer` that round-trips
  `model.Claims` through JSON + base64 instead of real ES256 signing —
  the promoted version of the old `fakeSigner`.

Both are never wired into `cmd/iamd/main.go` — they exist purely for
tests, but live under `internal/adapter` rather than inside
`internal/controller`'s test files because `internal/service` and
`internal/web` need them too, and a shared implementation beats three
copies of the same maps.

`internal/adapter/kubestore/store_test.go` keeps using a fake Kubernetes
clientset — it's testing `kubestore` itself, so faking anything below
that layer would defeat the point.

## Consequences

- `internal/controller`'s tests construct one `memorystore.New()` per
  test (it satisfies every port a test might need, the same
  single-store-many-ports shape `cmd/iamd/main.go` already uses for
  `kubestore.Store`) instead of one throwaway map-backed struct per port.
- `internal/service/service_test.go` and `internal/web/web_test.go` no
  longer import `k8s.io/client-go/kubernetes/fake`, `kubestore`, or
  `kubecrypt` at all — a `memorystore.New()`/`memorycrypto.Signer{}` pair
  stands in immediately, no `Load(ctx)` step needed.
- A future adapter (e.g. a real alternative `UserStore` for local dev
  without Kubernetes) has a second precedent for "add a sibling
  subpackage under internal/adapter" beyond the production ones.
