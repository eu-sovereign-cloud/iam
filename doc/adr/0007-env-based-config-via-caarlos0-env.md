# 0007 - Configuration via environment variables (caarlos0/env)

## Status

Accepted

## Context

`ecp`'s services configure themselves via cobra flags, not environment
variables. IAM deliberately does not follow that convention: as a
single-purpose, single-binary polyfill deployed into arbitrary self-install
clusters (often via a Helm chart or plain manifests), env-var configuration
is simpler to wire through a Pod spec than a flags-based CLI, and is the
prevailing convention for Kubernetes-native workloads generally.

## Decision

Use `github.com/caarlos0/env` to populate a single `config.Config` struct
from environment variables, with `envDefault` tags for optional settings
and `,required` for `IAM_JWT_ISSUER` (there is no sane default issuer URL).

## Consequences

- This is an intentional divergence from ecp's own convention; anyone
  moving between the two codebases should expect different config
  mechanisms, not assume shared tooling.
- No flags/CLI subcommands are needed since iamd exposes a single mode of
  operation (unlike ecp's gateway, which has `globalapiserver`/
  `regionalapiserver` subcommands).
