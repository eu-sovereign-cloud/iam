package model

import "time"

// User is an IAM-managed identity. Subject is the free-form external
// identifier (e.g. an email or a slug) stamped into issued JWTs as "sub".
type User struct {
	Subject     string
	DisplayName string
	Admin       bool
	CreatedAt   time.Time
}
