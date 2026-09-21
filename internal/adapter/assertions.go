package adapter

import "github.com/eu-sovereign-cloud/iam/internal/ports"

// Compile-time checks that the adapters here actually satisfy the ports
// internal/service depends on (ADR 0013) — catches a drifted method
// signature here rather than at first wiring in cmd/iamd/main.go.
var (
	_ ports.UserStore   = (*Store)(nil)
	_ ports.TenantStore = (*Store)(nil)
	_ ports.GrantStore  = (*Store)(nil)
	_ ports.PATStore    = (*Store)(nil)
	_ ports.Signer      = (*Signer)(nil)
	_ ports.Clock       = SystemClock{}
)
