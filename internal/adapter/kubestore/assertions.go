package kubestore

import "github.com/eu-sovereign-cloud/iam/internal/ports"

// Compile-time checks that Store actually satisfies the ports
// internal/controller depends on (ADR 0013, ADR 0014) — catches a drifted
// method signature here rather than at first wiring in cmd/iamd/main.go.
var (
	_ ports.UserStore   = (*Store)(nil)
	_ ports.TenantStore = (*Store)(nil)
	_ ports.GrantStore  = (*Store)(nil)
	_ ports.PATStore    = (*Store)(nil)
)
