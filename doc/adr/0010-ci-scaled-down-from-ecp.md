# 0010 - CI via plain GitHub Actions, scaled down from ecp

## Status

Accepted

## Context

`ecp`'s GitHub Actions setup is built for a very different shape of
project: a multi-module Go workspace with a custom published builder
container image (`ghcr.io/eu-sovereign-cloud/ecp-builder`, with the Go
toolchain version baked in via `.config.mk`'s `GO_VERSION`), a PR gate that
matrices `test`/`lint`/`gofmt`/`modernize`/`vuln`/`gosec` per changed Go
module (`pre-merge.yaml`), a KIND-based conformance suite
(`conformance.yaml`), and Helm chart release workflows
(`helm-release*.yaml`). `iam` is a single Go module producing one binary,
with no Helm chart and no conformance suite — there is nothing to matrix
over, and maintaining a separate builder image for one module would be
pure overhead with no payoff.

## Decision

Two workflows, using ordinary GitHub Actions building blocks instead of
ecp's bespoke container-image pipeline:

- `.github/workflows/ci.yml` — on every PR to `main` and push to `main`,
  one job runs `actions/setup-go` (`go-version-file: go.mod`, so the CI
  toolchain and the repo's declared Go version can't drift apart) and then
  `make build`, `make vet`, `make fmt-check`, `make test`,
  `golangci-lint-action`, and `make vuln` (a new Makefile target running
  `govulncheck`, the same relationship every other check already has to
  `make`). All in one job, sequentially — there's nothing to parallelize
  across for one module.
- `.github/workflows/image-release.yml` — on `v*` tags (or manual dispatch),
  builds the existing root `Dockerfile` and pushes
  `ghcr.io/eu-sovereign-cloud/iam:<tag>` using the built-in `GITHUB_TOKEN`,
  with a registry-based Buildx cache, matching ecp's `image-release.yaml`
  pattern (including its "no `:latest` tag" convention) minus the
  per-image matrix ecp needs for its four separate binaries.

Explicitly not carried over, because none of it applies to a single-module,
chart-less, conformance-suite-less repo: the custom builder image and its
publish/cleanup workflow, the module-diff/matrix machinery, the semantic-PR-title
check, chart-lint/chart-smoke, the conformance workflow, and the Helm
release workflows.

## Consequences

- Simpler to read and maintain: two short, ordinary-looking workflow files
  instead of ecp's ~7, with no shared container image to build, tag,
  version, or clean up.
- `golangci-lint`'s own `gosec` linter (already enabled in `.golangci.yml`)
  covers what ecp gets from a separate per-module `gosec` matrix cell; no
  standalone gosec job exists here because there's no matrix to hang it off
  of, not because the check itself was dropped.
- If `iam` later grows a second binary, a Helm chart, or a conformance
  suite, revisit this ADR — at that point some of ecp's machinery (a matrix,
  a chart-lint job, a conformance workflow) would start pulling its weight
  here too.
