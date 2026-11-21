package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func handleMemberAddedHistory(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.MemberAddedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal member.added history: %w", err)
		}
		performedBy := resolveActorID(p.PerformedBy, p.ActorID)
		roleID := p.RoleTemplateID
		roleName := p.RoleName
		record := &models.MembershipHistory{
			UserID:            p.UserID,
			OrganizationID:    &p.OrganizationID,
			Action:            models.MembershipHistoryActionAssigned,
			RoleTemplateID:    &roleID,
			RoleName:          &roleName,
			PerformedBy:       performedBy,
			OrganizationTitle: p.OrganizationTitle,
		}
		if err := insertMembershipHistory(deps,msg,record); err != nil {
			return fmt.Errorf("insert membership history: %w", err)
		}
		return nil
	}
}

func handleMemberUpdatedHistory(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.MemberUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal member.updated history: %w", err)
		}
		performedBy := resolveActorID(p.PerformedBy, p.ActorID)
		roleID := p.RoleTemplateID
		roleName := p.RoleName
		record := &models.MembershipHistory{
			UserID:            p.UserID,
			OrganizationID:    &p.OrganizationID,
			Action:            models.MembershipHistoryActionRoleChanged,
			RoleTemplateID:    &roleID,
			RoleName:          &roleName,
			PerformedBy:       performedBy,
			OrganizationTitle: p.OrganizationTitle,
		}
		if err := insertMembershipHistory(deps,msg,record); err != nil {
			return fmt.Errorf("insert membership history: %w", err)
		}
		return nil
	}
}

func handleMemberRemovedHistory(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.MemberRemovedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal member.removed history: %w", err)
		}
		performedBy := resolveActorID(p.PerformedBy, p.ActorID)
		roleID := p.RoleTemplateID
		roleName := p.RoleName
		record := &models.MembershipHistory{
			UserID:                 p.UserID,
			OrganizationID:         &p.OrganizationID,
			Action:                 models.MembershipHistoryActionRemoved,
			PreviousRoleTemplateID: &roleID,
			PreviousRoleName:       &roleName,
			PerformedBy:            performedBy,
			OrganizationTitle:      p.OrganizationTitle,
		}
		if err := insertMembershipHistory(deps,msg,record); err != nil {
			return fmt.Errorf("insert membership history: %w", err)
		}
		return nil
	}
}
