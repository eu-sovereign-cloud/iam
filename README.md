# iam

SECA IAM polyfill: a minimal, standalone user-management service that issues
JWTs compatible with the [ecp](https://github.com/eu-sovereign-cloud/ecp)
gateway's auth middleware, for deployments with no CSP-provided identity
provider (IONOS's self-installable model). See
[GitHub issue #1](https://github.com/eu-sovereign-cloud/iam/issues/1) for
the full ask, and `doc/adr/` for the architecture decisions behind this
implementation.

It manages **Users**, **Tenants**, and **Grants** (which tenants a user may
claim), and lets a user create/revoke **Personal Access Tokens (PATs)** —
which *are* signed JWTs, not a separate credential exchanged for one (see
ADR 0012). Only admins manage Tenants/Users/Grants; any user manages their
own PATs. Full IdP/SSO is out of scope — see issue #2 for the follow-on
OIDC discovery/JWKS/`/userinfo` work, which is also what will eventually
let `ecp` check whether a given PAT has been revoked (see ADR 0012's
accepted revocation-gap trade-off).

## Configuration

All configuration is via environment variables (`internal/config`):

| Variable                 | Default        | Notes                                   |
|---------------------------|----------------|------------------------------------------|
| `IAM_LISTEN_ADDR`          | `:8080`        |                                            |
| `IAM_NAMESPACE`            | `iam-system`   | Kubernetes namespace IAM stores state in  |
| `IAM_JWT_ISSUER`           | *(required)*   | `iss` claim on issued JWTs                |
| `IAM_JWT_AUDIENCE`         | *(empty)*      | `aud` claim, omitted if unset             |
| `KUBECONFIG`               | *(empty)*      | path to a kubeconfig; empty = in-cluster  |
| `IAM_LOG_LEVEL`            | `info`         |                                            |

## Running locally

Against a local cluster (e.g. `kind create cluster`):

```sh
export IAM_JWT_ISSUER=https://iam.example.com
export IAM_JWT_AUDIENCE=ecp-gateway
export KUBECONFIG=~/.kube/config
make run
```

On first startup, iamd creates an admin User and a bootstrap PAT, logging
the raw PAT once:

```
{"level":"WARN","msg":"created bootstrap admin PAT - copy it now, it will not be shown again","subject":"admin","pat":"iampat_..."}
```

### Example flow

```sh
ADMIN_PAT=iampat_...   # from the log line above

# Create a tenant
curl -s -X POST localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer $ADMIN_PAT" \
  -d '{"tenantId":"tenant-1","displayName":"Tenant One"}'

# Create a non-admin user and grant it that tenant
curl -s -X POST localhost:8080/api/v1/users \
  -H "Authorization: Bearer $ADMIN_PAT" \
  -d '{"subject":"alice@example.com","displayName":"Alice"}'
curl -s -X POST localhost:8080/api/v1/users/alice@example.com/grants \
  -H "Authorization: Bearer $ADMIN_PAT" \
  -d '{"tenantId":"tenant-1"}'

# Create a PAT for that user (self-service; here done with the admin PAT on their behalf).
# The "secret" returned is a signed JWT and IS the bearer credential — use
# it directly, there is no separate exchange step (ADR 0012).
curl -s -X POST localhost:8080/api/v1/users/alice@example.com/pats \
  -H "Authorization: Bearer $ADMIN_PAT" \
  -d '{"name":"laptop"}'
# => {"id":"...","subject":"alice@example.com",...,"secret":"eyJhbGci..."}
```

Decode the `secret` (a standard JWT) to see the claims shape the ecp
gateway expects (`sub`, `iss`, `aud`, `exp`, `tenants`, optional `scope`).

A minimal web UI is also available at `/web/login` (paste a PAT to sign in).

## Development

```sh
make build   # compile ./bin/iamd
make test    # go test ./... -race
make lint    # golangci-lint run
make fmt     # gofmt + goimports
make docker-build
```

## Architecture

See `doc/adr/` for the numbered architecture decision records (storage,
folder layout, JWT claims shape, revocation, signing key, bootstrap, config,
the Users/Tenants/Grants model, and the load-at-startup/no-watch/
single-replica constraint).
