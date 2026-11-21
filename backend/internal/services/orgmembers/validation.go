package orgmembers

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
	"time"
)

func parseMembershipExpiry(value *string, now time.Time) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	expiry, err := time.Parse(time.RFC3339, *value)
	if err != nil || !expiry.After(now) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("valid_until must be a future RFC3339 timestamp"))
	}
	return &expiry, nil
}
func validateRoleAssignment(role models.RoleTemplate, org models.Organization, holders int) error {
	if !role.IsActive || role.OrganizationTypeID != org.OrganizationTypeID || (role.OrganizationID != nil && *role.OrganizationID != org.ID) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role template must be active and belong to this organization or its type"))
	}
	if role.MaxHolders != nil && int(*role.MaxHolders) <= holders {
		return connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("role template has reached its maximum holders"))
	}
	return nil
}

// Serialize assignments on the role row so concurrent requests cannot exceed max_holders.
func validateAssignmentInTransaction(ctx context.Context, tx bun.Tx, orgID, roleID, excludeMemberID int32) error {
	var role models.RoleTemplate
	if err := tx.NewSelect().Model(&role).Where("id = ?", roleID).For("UPDATE").Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role template not found"))
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	var org models.Organization
	if err := tx.NewSelect().Model(&org).Where("id = ?", orgID).Scan(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	holders, err := tx.NewSelect().Model((*models.OrganizationMember)(nil)).Where("organization_id = ?", orgID).Where("role_template_id = ?", roleID).Where("id <> ?", excludeMemberID).Where("valid_until IS NULL OR valid_until > CURRENT_TIMESTAMP").Count(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	return validateRoleAssignment(role, org, holders)
}
