package kuberbac

import (
	"crypto/sha256"
	"encoding/hex"
)

// shortHash mirrors kubestore's own local helper (kubestore/names.go) —
// subjects may contain characters that aren't valid in a Kubernetes
// object name, so the RoleAssignment object name is a short hash of the
// subject, with the plaintext kept in a label for lookups.
func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

func roleAssignmentName(subject string) string {
	return "iam-" + shortHash(subject)
}
