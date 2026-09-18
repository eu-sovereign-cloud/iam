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
internal/service/       use cases + the ports (interfaces) they need from adapters
internal/adapter/       driven adapters implementing those ports (Kubernetes, JWT signing, hashing)
internal/controller/    driving adapter: JSON REST over net/http
internal/web/           driving adapter: server-rendered HTML
internal/config/        env-var configuration
```

Dependency direction: `controller` and `web` depend on `service`; `service`
depends on `model` and declares the ports it needs; `adapter` implements
those ports and depends on `model`. `model` depends on nothing else in the
module.

## Consequences

- Two "driving" adapters (`controller`, `web`) share the same `service`
  layer and never call each other, so business rules (e.g. only admins
  manage Tenants) are enforced once, not duplicated per UI.
- Unlike `ecp`, ports are declared in `service` rather than a separate
  `port` package, and there's a single Go module rather than `ecp`'s
  multi-module `framework`/`resource` split — proportionate to IAM's much
  smaller scope. `internal/model` having zero internal dependencies is the
  one boundary worth being strict about, so it alone gets a `depguard` rule
  in `.golangci.yml`.
