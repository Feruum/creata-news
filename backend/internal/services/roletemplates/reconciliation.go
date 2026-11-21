package roletemplates

import (
	"context"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func enqueueTemplateMemberSync(ctx context.Context, tx bun.Tx, templateID int32) error {
	var members []models.OrganizationMember
	if err := tx.NewSelect().Model(&members).Where("role_template_id = ?", templateID).Scan(ctx); err != nil {
		return err
	}
	for _, member := range members {
		if err := events.EnqueueMemberSync(ctx, tx, member.OrganizationID, member.UserID); err != nil {
			return err
		}
	}
	return nil
}
