package kuberbac

// Label keys set on every RoleAssignment IAM creates, so ListRoleAssignments
// (and repair's prune step) can find only IAM-managed objects and never
// touch a RoleAssignment created by anything else.
const (
	labelPrefix    = "iam.eu-sovereign-cloud/"
	labelManagedBy = labelPrefix + "managed-by"
	labelSubject   = labelPrefix + "subject"

	managedByGrant = "iam-grant"
)
