package adapter

import (
	"crypto/sha256"
	"encoding/hex"
)

// shortHash mirrors ecp's own namespace-hashing approach: subjects and
// tenant IDs are free-form strings that may not be valid Kubernetes object
// names, so the object name is a short hash of the natural key, with the
// plaintext kept in Data/annotations and a matching label for lookups.
func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

func userName(subject string) string {
	return "iam-user-" + shortHash(subject)
}

func tenantName(tenantID string) string {
	return "iam-tenant-" + shortHash(tenantID)
}

func grantName(subject, tenantID string) string {
	return "iam-grant-" + shortHash(subject) + "-" + shortHash(tenantID)
}

func patName(id string) string {
	return "iam-pat-" + id
}
