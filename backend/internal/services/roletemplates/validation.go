package roletemplates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func validateRoleTemplateUpdate(req *eventsv1.UpdateRoleTemplateRequest) error {
	if req.Id <= 0 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id must be positive"))
	}
	if req.ClearParentTemplate && req.ParentTemplateId != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("clear_parent_template conflicts with parent_template_id"))
	}
	if req.ClearMaxHolders && req.MaxHolders != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("clear_max_holders conflicts with max_holders"))
	}
	return validateRoleTemplateValues(req.ParentTemplateId, req.MaxHolders)
}

func validateRoleTemplateValues(parentID, maxHolders *int32) error {
	if parentID != nil && *parentID <= 0 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent_template_id must be positive"))
	}
	if maxHolders != nil && *maxHolders <= 0 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("max_holders must be positive"))
	}
	return nil
}

// The single-bigint "RTMP" namespace is separate from the two-int member locks.
// Serialize hierarchy changes for an organization type before taking row locks,
// so concurrent reciprocal parent changes cannot both validate a stale chain.
const roleTemplateHierarchyLockNamespace int64 = 0x52544D50 << 32

func lockRoleTemplateHierarchy(ctx context.Context, tx bun.Tx, organizationTypeID int32) error {
	key := roleTemplateHierarchyLockNamespace | int64(uint32(organizationTypeID))
	if _, err := tx.NewRaw("SELECT pg_advisory_xact_lock(?::bigint)", key).Exec(ctx); err != nil {
		return fmt.Errorf("lock role template hierarchy: %w", err)
	}
	return nil
}

func validateLockedParentTemplate(ctx context.Context, tx bun.Tx, child models.RoleTemplate, parentID *int32) error {
	return validateParentTemplate(child, parentID, func(id int32) (models.RoleTemplate, error) {
		var parent models.RoleTemplate
		err := tx.NewSelect().Model(&parent).
			Where("id = ?", id).
			Where("organization_type_id = ?", child.OrganizationTypeID).
			For("UPDATE").Scan(ctx)
		return parent, err
	})
}

func validateParentTemplate(child models.RoleTemplate, parentID *int32, lookup func(int32) (models.RoleTemplate, error)) error {
	seen := map[int32]bool{child.ID: true}
	for parentID != nil {
		if *parentID <= 0 {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent_template_id must be positive"))
		}
		if seen[*parentID] {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent template hierarchy contains a cycle"))
		}
		seen[*parentID] = true
		parent, err := lookup(*parentID)
		if errors.Is(err, sql.ErrNoRows) {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent template %d does not exist", *parentID))
		}
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("load parent template: %w", err))
		}
		if parent.OrganizationTypeID != child.OrganizationTypeID {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent template organization type must match the child"))
		}
		if parent.OrganizationID != nil && (child.OrganizationID == nil || *parent.OrganizationID != *child.OrganizationID) {
			return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parent template scope must be global or match the child's organization"))
		}
		child = parent
		parentID = parent.ParentTemplateID
	}
	return nil
}
