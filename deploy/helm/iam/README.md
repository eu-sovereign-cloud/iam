# iam Helm chart

Installs `iamd` (see the [top-level README](../../../README.md) and
`doc/adr/`) as a single-replica Deployment plus a Service, ServiceAccount,
and the RBAC it needs to manage its own state and ecp's `Role`/
`RoleAssignment` CRDs (see `doc/adr/0020`).

## Install

```sh
helm install iam oci://ghcr.io/eu-sovereign-cloud/charts/iam \
  --namespace iam-system --create-namespace \
  --set config.jwtIssuer=https://iam.example.com
```

`config.jwtIssuer` is required - it becomes the `IAM_JWT_ISSUER` env var
(the `iss` claim on every issued JWT) and there is no default, the same as
`internal/config.Config.JWTIssuer`.

## Values

See `values.yaml` for the full list; the ones most likely to need
overriding:

| Value                 | Default                              | Notes                                      |
|------------------------|---------------------------------------|----------------------------------------------|
| `config.jwtIssuer`     | *(required)*                          | `IAM_JWT_ISSUER`                              |
| `config.jwtAudience`   | `[]`                                  | `IAM_JWT_AUDIENCE`, comma-joined              |
| `config.namespace`     | release namespace                     | `IAM_NAMESPACE` - where iamd stores its state |
| `image.tag`            | chart's `appVersion`                  |                                                |
| `replicaCount`         | `1`                                   | fixed - see `doc/adr/0020`, `doc/adr/0009`    |

For the full environment variable reference, see the main README's
Configuration table.

## After install

iamd logs a one-time bootstrap admin PAT on first startup (ADR 0006):

```sh
kubectl logs -n iam-system deploy/iam-iam | grep 'bootstrap admin PAT'
```

## Uninstall

```sh
helm uninstall iam --namespace iam-system
```

This does not delete the `iam-signing-key` Secret's namespace or any
tenant namespaces `kuberbac` created - those are left in place, matching
how `helm uninstall` never touches resources it didn't create directly.
