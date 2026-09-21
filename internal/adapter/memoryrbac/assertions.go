package memoryrbac

import "github.com/eu-sovereign-cloud/iam/internal/ports"

var _ ports.TenantRoleStore = (*Store)(nil)
