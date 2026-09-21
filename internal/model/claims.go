package model

import "github.com/golang-jwt/jwt/v5"

// Claims is the JWT payload IAM issues. It matches, field-for-field, what
// ecp's gateway auth middleware (ecp/gateway/internal/authn/jwtstd.go)
// parses: standard registered claims plus the two SECA extensions "scope"
// and "tenants". See ADR 0003.
type Claims struct {
	jwt.RegisteredClaims
	Scope   *TokenScope `json:"scope,omitempty"`
	Tenants []string    `json:"tenants,omitempty"`
}
