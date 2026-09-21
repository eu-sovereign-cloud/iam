package model

// JWK is a single public signing key in JWKS format (RFC 7517), covering
// only the EC key parameters (RFC 7518 §6.2) IAM's ES256 key needs.
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

// JWKSet is the body of the /.well-known/jwks.json response (RFC 7517 §5).
type JWKSet struct {
	Keys []JWK `json:"keys"`
}
