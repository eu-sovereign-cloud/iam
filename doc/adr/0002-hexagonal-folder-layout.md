# 0002 - Hexagonal folder layout

## Status

Accepted

## Context

IAM needs a folder structure that keeps its business logic (who may create
a PAT, what a JWT's claims look like) independent from how it's delivered
over HTTP and how it's persisted, so either side can change without
touching the other — e.g. swapping the ConfigMap/Secret store for something
else later, or adding a gRPC API alongside REST, without rewriting the use
cases in between.

## Decision

Single Go module, hexagonal-flavored layout:

```
cmd/iamd/main.go        wiring only
internal/model/         domain types, no dependencies on any other internal package
internal/ports/         port interfaces (see ADR 0013), no dependencies beyond model
internal/service/       use cases, depending on model + ports
internal/adapter/       driven adapters implementing those ports (Kubernetes, JWT signing, hashing)
internal/controller/    driving adapter: JSON REST over net/http
internal/web/           driving adapter: server-rendered HTML
internal/config/        env-var configuration
```

Dependency direction: `controller` and `web` depend on `service`; `service`
depends on `model` and `ports`; `adapter` implements the interfaces `ports`
declares and depends on `model`. `model` and `ports` depend on nothing else
in the module (see ADR 0013 for why ports moved out of `service`).

## Consequences

- Two "driving" adapters (`controller`, `web`) share the same `service`
  layer and never call each other, so business rules (e.g. only admins
  manage Tenants) are enforced once, not duplicated per UI.
- ~~Unlike `ecp`, ports are declared in `service` rather than a separate
  `port` package~~ — reversed by ADR 0013 once the number of ports and
  their surrounding services grew past what one shared file stayed tidy
  for; ports now live in their own `internal/ports` package, one file per
  port, with the same "zero internal dependencies" `depguard` rule
  `internal/model` gets. There's still a single Go module rather than
  `ecp`'s multi-module `framework`/`resource` split — proportionate to
  IAM's much smaller scope.
