package model

// TokenScope is an optional down-scoping cap a PAT can carry. It mirrors
// ecp's resource.TokenScope wire format field-for-field (see ADR 0003) so a
// PAT's scope can be copied verbatim into the "scope" claim of an issued
// JWT: base64(JSON{...,"scope":{"tenants":[...],"regions":[...],"workspaces":[...]}}).
type TokenScope struct {
	Tenants    []string `json:"tenants,omitempty"`
	Regions    []string `json:"regions,omitempty"`
	Workspaces []string `json:"workspaces,omitempty"`
}
