package service

import (
	"context"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

// GrantService manages a User's access to Tenants. Callers (the controller
// layer) are responsible for enforcing that only admins invoke the
// mutating methods.
//
// TODO(eu-sovereign-cloud/iam#future): when a Grant is created or deleted,
// this is the place to eventually notify ecp so it can create/remove the
// corresponding RoleAssignment in the tenant's namespace. Not implemented:
// IAM has no ecp credentials/access yet and the IAM-grant -> ecp-role
// mapping isn't defined. See ADR 0008.
type GrantService struct {
	grants  GrantStore
	users   UserStore
	tenants TenantStore
	clock   Clock
}

func NewGrantService(grants GrantStore, users UserStore, tenants TenantStore, clock Clock) *GrantService {
	return &GrantService{grants: grants, users: users, tenants: tenants, clock: clock}
}

func (s *GrantService) Create(ctx context.Context, subject, tenantID, grantedBy string) (model.Grant, error) {
	if _, err := s.users.GetUser(ctx, subject); err != nil {
		return model.Grant{}, err
	}
	if _, err := s.tenants.GetTenant(ctx, tenantID); err != nil {
		return model.Grant{}, err
	}
	g := model.Grant{
		Subject:   subject,
		TenantID:  tenantID,
		GrantedAt: s.clock.Now(),
		GrantedBy: grantedBy,
	}
	if err := s.grants.CreateGrant(ctx, g); err != nil {
		return model.Grant{}, err
	}
	return g, nil
}

func (s *GrantService) ListBySubject(ctx context.Context, subject string) ([]model.Grant, error) {
	return s.grants.ListGrantsBySubject(ctx, subject)
}

func (s *GrantService) Delete(ctx context.Context, subject, tenantID string) error {
	return s.grants.DeleteGrant(ctx, subject, tenantID)
}
