package roletemplates

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) CreateRoleTemplate(ctx context.Context, req *connect.Request[eventsv1.CreateRoleTemplateRequest]) (*connect.Response[eventsv1.CreateRoleTemplateResponse], error) {
	slog.Debug("CreateRoleTemplate", "key", req.Msg.Key, "displayName", req.Msg.DisplayName)

	if req.Msg.OrganizationTypeId <= 0 || (req.Msg.OrganizationId != nil && *req.Msg.OrganizationId <= 0) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization type and organization ids must be positive"))
	}
	if err := validateRoleTemplateValues(req.Msg.ParentTemplateId, req.Msg.MaxHolders); err != nil {
		return nil, err
	}

	// Authorization: require manage_settings on the organization, or platform staff for org-type defaults
	if req.Msg.OrganizationId != nil {
		// Organization-specific role template
		if _, err := s.authz.RequireOrgPermission(ctx, *req.Msg.OrganizationId, "manage_settings"); err != nil {
			return nil, err
		}
	} else {
		// Default role template for org type - requires platform staff
		if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
			return nil, err
		}
	}

	template := models.RoleTemplate{
		OrganizationTypeID: req.Msg.OrganizationTypeId,
		Key:                req.Msg.Key,
		DisplayName:        req.Msg.DisplayName,
		SortOrder:          req.Msg.SortOrder,
		IsRequired:         req.Msg.IsRequired,
		IsActive:           req.Msg.IsActive,
	}

	if req.Msg.OrganizationId != nil {
		template.OrganizationID = req.Msg.OrganizationId
	}
	if req.Msg.Description != nil {
		template.Description = req.Msg.Description
	}
	if req.Msg.ParentTemplateId != nil {
		template.ParentTemplateID = req.Msg.ParentTemplateId
	}
	if req.Msg.MaxHolders != nil {
		template.MaxHolders = req.Msg.MaxHolders
	}

	caps, err := validatedCapabilities(req.Msg.Capabilities)
	if err != nil {
		return nil, err
	}
	if template.OrganizationID != nil {
		var org models.Organization
		if err := s.db.NewSelect().Model(&org).Where("id = ?", *template.OrganizationID).Scan(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if org.OrganizationTypeID != template.OrganizationTypeID {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization type does not match role template"))
		}
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	if template.ParentTemplateID != nil {
		if err := lockRoleTemplateHierarchy(ctx, tx, template.OrganizationTypeID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := validateLockedParentTemplate(ctx, tx, template, template.ParentTemplateID); err != nil {
			return nil, err
		}
	}
	if _, err := tx.NewInsert().Model(&template).Returning("*").Exec(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for i := range caps {
		caps[i].RoleTemplateID = template.ID
	}
	if len(caps) > 0 {
		if _, err := tx.NewInsert().Model(&caps).Exec(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&eventsv1.CreateRoleTemplateResponse{
		RoleTemplate: dbRoleTemplateToProto(template, caps),
	}), nil
}
