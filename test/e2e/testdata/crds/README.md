# Vendored ecp CRDs

`authorization.v1.secapi.cloud_roles.yaml` and
`authorization.v1.secapi.cloud_role-assignments.yaml` are verbatim copies of
the same files from `eu-sovereign-cloud/ecp`'s
`charts/ecp/crds/` directory, at commit
`5f180aa61cc44ff7903548b541bb2daebc275fe1`.

They are **not authored here**. ecp generates them via `controller-gen` from
the kubebuilder markers on `resource/authorization/v1/role/backend/kubernetes/resource.go`
and `resource/authorization/v1/role-assignment/backend/kubernetes/resource.go`
(see ecp's `framework/backend/kubernetes/Makefile` `generate-crds` target);
`charts/ecp/crds/` is the single directory that generation writes to, and
it's also what ecp's own `envtest`-based Go tests load CRDs from
(`setup_envtest_test.go` in both of those directories), so vendoring these
exact files is "the same thing ecp uses," not a hand-rolled substitute.

## Why these two, and why here at all

IAM itself never creates or reads `Role`/`RoleAssignment` objects — see ADR
0008: creating the RoleAssignment that should eventually correspond to an
IAM Grant is explicit, unimplemented future work. These CRDs exist in the
e2e test purely to confirm that a `RoleAssignment` shaped exactly the way
that future integration would produce is still accepted by ecp's schema —
a regression check on *this vendored copy*, not on IAM's own behavior. See
`doc/adr/0011-e2e-test-against-kind.md`.

## Refreshing

If ecp's `Role`/`RoleAssignment` schema changes, re-copy these two files
from the current `charts/ecp/crds/` in `eu-sovereign-cloud/ecp` and update
the commit hash above. There is no automation keeping this in sync;
drift here would only be caught by the e2e test failing to apply an
example `RoleAssignment` (see `test/e2e/e2e_test.go`), not by any active
comparison against ecp's repo.
