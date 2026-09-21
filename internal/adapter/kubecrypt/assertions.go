package kubecrypt

import "github.com/eu-sovereign-cloud/iam/internal/ports"

// Compile-time check that Signer actually satisfies the port
// internal/controller depends on (ADR 0013, ADR 0014) — catches a drifted
// method signature here rather than at first wiring in cmd/iamd/main.go.
var _ ports.Signer = (*Signer)(nil)
