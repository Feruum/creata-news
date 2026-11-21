package orgmembers

import (
	"context"
	"fmt"
	"github.com/uptrace/bun"
	"log/slog"
	"time"

	"github.com/studyverse/ems-backend/internal/models"
)

// RecordHistoryParams holds all fields for a membership history entry.
type RecordHistoryParams struct {
	UserID                 int32
	OrganizationID         int32
	Action                 models.MembershipHistoryAction
	RoleTemplateID         *int32
	PreviousRoleTemplateID *int32
	PerformedBy            *int32
	Reason                 *string
	RoleName               *string
	PreviousRoleName       *string
	OrganizationTitle      string
	EffectiveAt            *time.Time // nil = now
}

// recordMembershipHistory inserts a single history row.
// Non-fatal: logs error but does not fail the parent mutation.
func (s *Service) recordMembershipHistory(ctx context.Context, p RecordHistoryParams) {
	effectiveAt := time.Now()
	if p.EffectiveAt != nil {
		effectiveAt = *p.EffectiveAt
	}
	record := &models.MembershipHistory{
		UserID:                 p.UserID,
		OrganizationID:         &p.OrganizationID,
		Action:                 p.Action,
		RoleTemplateID:         p.RoleTemplateID,
		PreviousRoleTemplateID: p.PreviousRoleTemplateID,
		PerformedBy:            p.PerformedBy,
		Reason:                 p.Reason,
		RoleName:               p.RoleName,
		PreviousRoleName:       p.PreviousRoleName,
		OrganizationTitle:      p.OrganizationTitle,
		EffectiveAt:            effectiveAt,
	}
	if _, err := s.db.NewInsert().Model(record).Exec(ctx); err != nil {
		slog.Error("failed to record membership history",
			"user_id", p.UserID,
			"org_id", p.OrganizationID,
			"action", p.Action,
			"error", err,
		)
	}
}

// memberHistoryRow is a joined struct for bulk queries.
type memberHistoryRow struct {
	UserID         int32  `bun:"user_id"`
	RoleTemplateID int32  `bun:"role_template_id"`
	RoleName       string `bun:"role_name"`
}

// RecordBulkMembershipHistory inserts history rows for ALL current members of an org.
// Used before org delete or archive. Non-fatal: logs errors.
func (s *Service) RecordBulkMembershipHistory(ctx context.Context, orgID int32, orgTitle string, action models.MembershipHistoryAction, performedBy *int32) {
	if err := RecordBulkMembershipHistoryInTransaction(ctx, s.db, orgID, orgTitle, action, performedBy); err != nil {
		slog.ErrorContext(ctx, "failed to record bulk membership history", "org_id", orgID, "action", action, "error", err)
	}
}

// RecordBulkMembershipHistoryInTransaction keeps history and the organization mutation atomic.
func RecordBulkMembershipHistoryInTransaction(ctx context.Context, tx bun.IDB, orgID int32, orgTitle string, action models.MembershipHistoryAction, performedBy *int32) error {
	var members []memberHistoryRow
	if err := tx.NewSelect().TableExpr("organization_members AS om").ColumnExpr("om.user_id").ColumnExpr("om.role_template_id").ColumnExpr("rt.display_name AS role_name").Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").Where("om.organization_id = ?", orgID).Scan(ctx, &members); err != nil {
		return fmt.Errorf("load organization members for history: %w", err)
	}
	if len(members) == 0 {
		return nil
	}
	records := make([]models.MembershipHistory, 0, len(members))
	now := time.Now()
	for _, member := range members {
		roleName := member.RoleName
		roleID := member.RoleTemplateID
		records = append(records, models.MembershipHistory{UserID: member.UserID, OrganizationID: &orgID, Action: action, PreviousRoleTemplateID: &roleID, PreviousRoleName: &roleName, PerformedBy: performedBy, OrganizationTitle: orgTitle, EffectiveAt: now})
	}
	if _, err := tx.NewInsert().Model(&records).Exec(ctx); err != nil {
		return fmt.Errorf("insert bulk membership history: %w", err)
	}
	return nil
}
