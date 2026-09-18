package model

import "time"

// Tenant is a SECA tenant IAM knows about. TenantID is the free-form
// identifier that ends up in a JWT's "tenants" claim and, on the ecp side,
// gets hashed into a namespace name.
type Tenant struct {
	TenantID    string
	DisplayName string
	CreatedAt   time.Time
}
