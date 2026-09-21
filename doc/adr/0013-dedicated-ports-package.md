# 0013 - Dedicated `internal/ports` package, one file per port

## Status

Accepted. Reverses the specific choice ADR 0002 made to keep ports inside
`service`; everything else in ADR 0002 (the overall hexagonal layering,
dependency direction) still stands.

## Context

Ports (`UserStore`, `TenantStore`, `GrantStore`, `PATStore`, `Signer`,
`Clock`) originally all lived in one file, `internal/service/ports.go`.
ADR 0002 chose that deliberately — `service` was small enough that a
separate `port` package (the way `ecp`'s `framework/kernel/port` does it)
would have been one more package for little payoff. That stopped being
true: six ports, five services (`internal/service/*_service.go`), and a
growing set of ADRs referencing individual ports made one undifferentiated
file harder to navigate than it needed to be.

## Decision

- `internal/ports`, sitting alongside `internal/model` as a second
  dependency-free leaf package (only imports `model` + stdlib) — one file
  per port: `user_store.go`, `tenant_store.go`, `grant_store.go`,
  `pat_store.go`, `signer.go`, `clock.go`, plus a `doc.go` for the package
  comment.
- `SystemClock` (the concrete `time.Now()`-backed implementation of
  `Clock`) moves to `internal/adapter/clock.go` — it's a driven
  implementation, not the port itself, so it belongs with `Store` and
  `Signer`, not in the now interface-only `ports` package.
- `internal/service`'s five service structs import `ports` and reference
  the qualified names (`ports.UserStore`, `ports.Clock`, etc.) instead of
  bare ones. No behavior changes — same fields, same method bodies.
- `internal/adapter` gets a small `assertions.go` with compile-time
  satisfaction checks (`var _ ports.UserStore = (*Store)(nil)`, etc.) —
  cheap, and catches a drifted method signature at build time instead of
  at first wiring in `cmd/iamd/main.go`. Adapters still don't *need* to
  import `ports` to satisfy it structurally; this import exists purely for
  that assertion.
- `.golangci.yml`'s `depguard` gets a second rule mirroring `model`'s:
  `internal/ports` may not import `adapter`/`service`/`controller`/`web`/
  `config`, keeping it a leaf like `model`.

## Consequences

- Test doubles (`internal/service/fakes_test.go`) needed **no changes** —
  Go interfaces are structurally satisfied, so a fake store already
  satisfies `ports.UserStore` without ever naming or importing that type.
  This is a nice confirmation that the port/adapter boundary was already
  clean; moving the interfaces' *location* didn't require touching any of
  their implementers.
- One more package to keep in view when reading the codebase end to end,
  in exchange for each port being a single, easy-to-find small file rather
  than a shared one growing indefinitely as more ports get added.
