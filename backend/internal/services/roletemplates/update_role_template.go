package roletemplates

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) UpdateRoleTemplate(ctx context.Context, req *connect.Request[eventsv1.UpdateRoleTemplateRequest]) (*connect.Response[eventsv1.UpdateRoleTemplateResponse], error) {
	slog.Debug("UpdateRoleTemplate", "id", req.Msg.Id)

	if err := validateRoleTemplateUpdate(req.Msg); err != nil {
		return nil, err
	}

	var existing models.RoleTemplate
	err := s.db.NewSelect().Model(&existing).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if existing.OrganizationID != nil {
		if _, permErr := s.authz.RequireOrgPermission(ctx, *existing.OrganizationID, "manage_settings"); permErr != nil {
			return nil, permErr
		}
	} else {
		if _, permErr := s.authz.RequirePlatformStaff(ctx); permErr != nil {
			return nil, permErr
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	if req.Msg.ParentTemplateId != nil || req.Msg.ClearParentTemplate {
		if err := lockRoleTemplateHierarchy(ctx, tx, existing.OrganizationTypeID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if err := tx.NewSelect().Model(&existing).Where("id = ?", req.Msg.Id).For("UPDATE").Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if req.Msg.ParentTemplateId != nil {
		if err := validateLockedParentTemplate(ctx, tx, existing, req.Msg.ParentTemplateId); err != nil {
			return nil, err
		}
	}

	template := existing
	if columns := applyRoleTemplateUpdate(&template, req.Msg); len(columns) > 0 {
		err = tx.NewUpdate().Model(&template).Column(columns...).WherePK().Returning("*").Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, nil)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	var caps []models.RoleTemplateCapability
	err = tx.NewSelect().Model(&caps).Where("role_template_id = ?", template.ID).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if req.Msg.IsActive != nil {
		if err := enqueueTemplateMemberSync(ctx, tx, template.ID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if req.Msg.IsActive != nil {
		if err := s.syncTemplateMembers(ctx, template.ID); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
	}

	return connect.NewResponse(&eventsv1.UpdateRoleTemplateResponse{
		RoleTemplate: dbRoleTemplateToProto(template, caps),
	}), nil
}

func applyRoleTemplateUpdate(template *models.RoleTemplate, req *eventsv1.UpdateRoleTemplateRequest) []string {
	var columns []string
	if req.DisplayName != nil {
		template.DisplayName = *req.DisplayName
		columns = append(columns, "display_name")
	}
	if req.Description != nil {
		template.Description = req.Description
		columns = append(columns, "description")
	}
	if req.ParentTemplateId != nil || req.ClearParentTemplate {
		template.ParentTemplateID = req.ParentTemplateId
		columns = append(columns, "parent_template_id")
	}
	if req.SortOrder != nil {
		template.SortOrder = *req.SortOrder
		columns = append(columns, "sort_order")
	}
	if req.MaxHolders != nil || req.ClearMaxHolders {
		template.MaxHolders = req.MaxHolders
		columns = append(columns, "max_holders")
	}
	if req.IsRequired != nil {
		template.IsRequired = *req.IsRequired
		columns = append(columns, "is_required")
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
		columns = append(columns, "is_active")
	}
	return columns
}
