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

func (s *Service) SetTemplateCapabilities(ctx context.Context, req *connect.Request[eventsv1.SetTemplateCapabilitiesRequest]) (*connect.Response[eventsv1.SetTemplateCapabilitiesResponse], error) {
	slog.Debug("SetTemplateCapabilities", "templateId", req.Msg.RoleTemplateId, "capCount", len(req.Msg.Capabilities))

	var existing models.RoleTemplate
	err := s.db.NewSelect().Model(&existing).Where("id = ?", req.Msg.RoleTemplateId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if existing.OrganizationID != nil {
		if _, err := s.authz.RequireOrgPermission(ctx, *existing.OrganizationID, "manage_settings"); err != nil {
			return nil, err
		}
	} else {
		if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
			return nil, err
		}
	}

	caps, err := validatedCapabilities(req.Msg.Capabilities)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.NewSelect().Model(&existing).Where("id = ?", req.Msg.RoleTemplateId).For("UPDATE").Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if _, err := tx.NewDelete().Model((*models.RoleTemplateCapability)(nil)).Where("role_template_id = ?", req.Msg.RoleTemplateId).Exec(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for i := range caps {
		caps[i].RoleTemplateID = existing.ID
	}
	if len(caps) > 0 {
		if _, err := tx.NewInsert().Model(&caps).Exec(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if err := enqueueTemplateMemberSync(ctx, tx, existing.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.syncTemplateMembers(ctx, existing.ID); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	template := existing
	return connect.NewResponse(&eventsv1.SetTemplateCapabilitiesResponse{
		RoleTemplate: dbRoleTemplateToProto(template, caps),
	}), nil
}
