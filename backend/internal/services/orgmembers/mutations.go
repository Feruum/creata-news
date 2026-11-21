package orgmembers

import (
	"context"
	"fmt"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) enqueueMembershipMutation(ctx context.Context, tx bun.Tx, member models.OrganizationMember, actorID int32, kind string) error {
	if err := events.EnqueueMemberSync(ctx, tx, member.OrganizationID, member.UserID); err != nil {
		return err
	}
	publisher := s.publisher.InTransaction(tx)
	if publisher == nil {
		return nil
	}
	var orgTitle, roleName string
	if err := tx.NewSelect().Model((*models.Organization)(nil)).Column("title").Where("id = ?", member.OrganizationID).Scan(ctx, &orgTitle); err != nil {
		return fmt.Errorf("load membership organization: %w", err)
	}
	if err := tx.NewSelect().Model((*models.RoleTemplate)(nil)).Column("display_name").Where("id = ?", member.RoleTemplateID).Scan(ctx, &roleName); err != nil {
		return fmt.Errorf("load membership role: %w", err)
	}
	switch kind {
	case "added":
		return publisher.PublishMemberAdded(ctx, events.MemberAddedPayload{UserID: member.UserID, OrganizationID: member.OrganizationID, OrganizationTitle: orgTitle, RoleTemplateID: member.RoleTemplateID, RoleName: roleName, PerformedBy: &actorID, ActorID: &actorID})
	case "updated":
		return publisher.PublishMemberUpdated(ctx, events.MemberUpdatedPayload{UserID: member.UserID, OrganizationID: member.OrganizationID, OrganizationTitle: orgTitle, RoleTemplateID: member.RoleTemplateID, RoleName: roleName, PerformedBy: &actorID, ActorID: &actorID})
	case "removed":
		return publisher.PublishMemberRemoved(ctx, events.MemberRemovedPayload{UserID: member.UserID, OrganizationID: member.OrganizationID, OrganizationTitle: orgTitle, RoleTemplateID: member.RoleTemplateID, RoleName: roleName, PerformedBy: &actorID, ActorID: &actorID})
	default:
		return fmt.Errorf("unknown membership mutation: %s", kind)
	}
}
