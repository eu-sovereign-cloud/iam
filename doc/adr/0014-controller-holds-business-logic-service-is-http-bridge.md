# 0014 - `controller` holds business logic, `service` is the HTTP/JSON bridge

## Status

Accepted. Reverses the specific role assignment ADR 0002 gave to `service`
and `controller`; the overall hexagonal layering and dependency direction
still stand, just with those two package names swapped.

## Context

Since ADR 0002, `internal/service` held business logic grouped by resource
(`UserService`, `TenantService`, `GrantService`, `PATService`,
`AuthService`, each with several methods), and `internal/controller` was
the JSON/HTTP driving adapter (routing, request decoding, response
encoding) that called into `internal/service`. That reads backwards from
the more common Go convention, where a "controller" holds use-case logic
and a "service" is closer to the driving/delivery layer — worth fixing
before the naming calcifies further under more ADRs and call sites.

While making the swap, two more changes were folded in:

- One controller struct per operation (`CreateTenant`, `DeleteUser`,
  `AuthenticatePAT`, ...) instead of one struct per resource with several
  methods. Each holds only the `ports.*` (and, where needed, config) its
  own logic actually needs as constructor-injected fields — most need one
  or two, not a resource's full dependency set. Every one is invoked via a
  uniform `Do(ctx, ...) (..., error)` method.
- `identityContextKey`/`withIdentity`/`identityFromContext` moved from the
  REST layer to `internal/model/context.go` (exported as `WithIdentity`/
  `IdentityFromContext`), absorbing `internal/web`'s own near-duplicate
  copy of the same three functions. Both driving adapters now share one
  implementation, keeping the REST layer's stricter behavior: panic if
  identity is read without going through a `RequireAuth`/`requireAuth`
  wrapper first, since both adapters' handlers are always wrapped by one.

## Decision

- `internal/controller` is the business-logic layer: one file per
  operation (snake_case, matching the `internal/ports` precedent from ADR
  0013), no mention of HTTP/JSON/presentation anywhere, working only with
  `model.*` types, `context.Context`, and `ports.*` dependencies. 19
  controllers total, covering every operation the two driving adapters
  need (tenants, users, grants, PATs, PAT/user authentication, and
  bootstrap-admin provisioning). Controllers may depend on other
  controllers where that avoids duplicating logic (`AuthenticateUser`
  wraps `AuthenticatePAT`; `EnsureBootstrapAdmin` composes
  `ListUsers`+`CreateUser`+`CreatePAT`) — this is ordinary use-case
  composition, not a layering violation.
- `internal/service` is the JSON/HTTP bridge: routing, `encoding/json`,
  `Authorization` header parsing, status-code mapping — today's
  `internal/controller` relocated and renamed, calling into the new
  `internal/controller` package for the actual work. Its top-level wiring
  struct is `service.Service` (mirroring `web.Web` — one wiring struct per
  driving-adapter package), holding pointers to only the individual
  controllers its handlers call, not whole resource groups.
- `internal/web` keeps its package name (already unambiguous — HTML, not
  JSON) but switches from depending on `internal/service` to depending on
  `internal/controller` directly, the same as `internal/service` does.
  Its `Web` struct follows the same per-controller-pointer shape as
  `service.Service`.
- `internal/model/context.go` holds the shared identity-in-context
  helpers; `internal/controller/ids.go` (the `newID()` UUID helper) moved
  alongside the controllers that use it, as controller-internal plumbing
  rather than a port.

## Consequences

- The dependency direction from ADR 0002 is unchanged, only the names at
  each end of the arrow: `service` and `web` (driving adapters) depend on
  `controller` (business logic); `controller` depends on `model` and
  `ports`; `adapter` implements `ports` and depends on `model`.
- Test doubles (`internal/controller/fakes_test.go`, moved verbatim from
  the old `internal/service/fakes_test.go`) needed no interface-shape
  changes, the same structural-typing win ADR 0013 already noted when
  ports moved packages.
- Production code is one-controller-per-file, but tests are grouped by
  resource area for navigability (`tenant_test.go`, `user_test.go`,
  `grant_test.go`, `pat_test.go`, `auth_test.go`, `bootstrap_test.go`) —
  a test-file-organization choice, not a statement about the controllers
  themselves.
- `internal/service/service_test.go` replaces the old
  `internal/controller/controller_test.go` as the full-stack HTTP/fake-k8s
  integration test; `internal/web/web_test.go` keeps the same test bodies,
  updated only for the new constructor/wiring shape.
