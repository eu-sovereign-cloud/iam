package kubestore

// Label and annotation keys used on the ConfigMaps/Secrets IAM manages
// (see ADR 0001 and the data-model section of the design plan).
const (
	labelPrefix = "iam.eu-sovereign-cloud/"

	labelType   = labelPrefix + "type"
	labelAdmin  = labelPrefix + "admin"
	labelUser   = labelPrefix + "user"
	labelTenant = labelPrefix + "tenant"

	typeUser   = "user"
	typeTenant = "tenant"
	typeGrant  = "grant"
	typePAT    = "pat"

	dataSubject     = "subject"
	dataDisplayName = "display-name"
	dataAdmin       = "admin"
	dataCreatedAt   = "created-at"

	dataTenantID = "tenant-id"

	dataGrantedAt = "granted-at"
	dataGrantedBy = "granted-by"

	dataName      = "name"
	dataScope     = "scope"
	dataExpiresAt = "expires-at"
)
