package orgmembers

import (
	"context"
)

func (s *Service) syncMemberCapabilitiesToSpiceDB(ctx context.Context, orgID, userID, roleTemplateID int32) error {
	if s.perms == nil {
		return nil
	}
	return s.authz.SyncOrganizationMember(ctx, orgID, userID)
}
func (s *Service) removeMemberFromSpiceDB(ctx context.Context, orgID, userID int32) error {
	if s.perms == nil {
		return nil
	}
	return s.authz.SyncOrganizationMember(ctx, orgID, userID)
}
