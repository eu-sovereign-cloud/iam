package kuberbac

import "k8s.io/apimachinery/pkg/runtime/schema"

// The CRD group/version and the two resources this adapter manages. ecp
// defines these in resource/authorization/v1/{role,role-assignment}/domain.go
// — duplicated here as literals since that package isn't importable (see
// the package doc comment).
const (
	group   = "authorization.v1.secapi.cloud"
	version = "v1"
)

var (
	roleGVR           = schema.GroupVersionResource{Group: group, Version: version, Resource: "roles"}
	roleAssignmentGVR = schema.GroupVersionResource{Group: group, Version: version, Resource: "role-assignments"}
)

// permission and roleSpec mirror ecp's Role.Spec (RoleSpec{Permissions
// []Permission}) field-for-field — see
// test/e2e/testdata/crds/authorization.v1.secapi.cloud_roles.yaml for the
// vendored schema this must stay compatible with.
type permission struct {
	Provider  string   `json:"provider"`
	Resources []string `json:"resources"`
	Verb      []string `json:"verb"`
}

type roleSpec struct {
	Permissions []permission `json:"permissions"`
}

// roleAssignmentScope and roleAssignmentSpec mirror ecp's
// RoleAssignment.Spec (RoleAssignmentSpec{Subs, Roles []string, Scopes
// []RoleAssignmentScope}) — see
// test/e2e/testdata/crds/authorization.v1.secapi.cloud_role-assignments.yaml.
type roleAssignmentScope struct {
	Tenants    []string `json:"tenants,omitempty"`
	Regions    []string `json:"regions,omitempty"`
	Workspaces []string `json:"workspaces,omitempty"`
}

type roleAssignmentSpec struct {
	Subs   []string              `json:"subs"`
	Scopes []roleAssignmentScope `json:"scopes"`
	Roles  []string              `json:"roles"`
}
