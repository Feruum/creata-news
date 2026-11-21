package roletemplates

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func validatedCapabilities(input []eventsv1.Capability) ([]models.RoleTemplateCapability, error) {
	seen := map[models.Capability]bool{}
	caps := make([]models.RoleTemplateCapability, 0, len(input))
	for _, cap := range input {
		value := protoCapabilityToDB(cap)
		if value == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown capability"))
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		caps = append(caps, models.RoleTemplateCapability{Capability: value})
	}
	return caps, nil
}
func (s *Service) syncTemplateMembers(ctx context.Context, templateID int32) error {
	var members []models.OrganizationMember
	if err := s.db.NewSelect().Model(&members).Where("role_template_id = ?", templateID).Scan(ctx); err != nil {
		return fmt.Errorf("load template members: %w", err)
	}
	var firstErr error
	for _, member := range members {
		if err := s.authz.SyncOrganizationMember(ctx, member.OrganizationID, member.UserID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
