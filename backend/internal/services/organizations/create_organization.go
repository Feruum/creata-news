package organizations

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/slug"
)

func (s *Service) CreateOrganization(ctx context.Context, req *connect.Request[eventsv1.CreateOrganizationRequest]) (*connect.Response[eventsv1.CreateOrganizationResponse], error) {
	slog.Debug("CreateOrganization", "title", req.Msg.Title)

	localUserID, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	baseSlug := slug.Slugify(req.Msg.Title)
	orgSlug := baseSlug
	for i := 1; i <= 100; i++ {
		exists, slugErr := tx.NewSelect().Model((*models.OrganizationSlug)(nil)).Where("slug = ?", orgSlug).Exists(ctx)
		if slugErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check slug: %w", slugErr))
		}
		if !exists {
			break
		}
		orgSlug = slug.WithSuffix(baseSlug, i+1)
	}

	status := protoStatusToDB(req.Msg.Status)
	org := models.Organization{
		Title:              req.Msg.Title,
		OrganizationTypeID: req.Msg.OrganizationTypeId,
		Status:             &status,
	}

	if req.Msg.ImageUrl != nil {
		org.ImageUrl = req.Msg.ImageUrl
	}
	if req.Msg.Description != nil {
		org.Description = json.RawMessage(*req.Msg.Description)
	}
	if req.Msg.Instagram != nil {
		org.Instagram = req.Msg.Instagram
	}
	if req.Msg.TelegramChannel != nil {
		org.TelegramChannel = req.Msg.TelegramChannel
	}
	if req.Msg.TelegramChat != nil {
		org.TelegramChat = req.Msg.TelegramChat
	}
	if req.Msg.Website != nil {
		org.Website = req.Msg.Website
	}
	if req.Msg.Youtube != nil {
		org.Youtube = req.Msg.Youtube
	}
	if req.Msg.Tiktok != nil {
		org.Tiktok = req.Msg.Tiktok
	}
	if req.Msg.Linkedin != nil {
		org.Linkedin = req.Msg.Linkedin
	}

	_, err = tx.NewInsert().Model(&org).Returning("*").Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	orgSlugModel := models.OrganizationSlug{
		OrganizationID: org.ID,
		Slug:           orgSlug,
		IsCurrent:      true,
	}
	if _, slugInsertErr := tx.NewInsert().Model(&orgSlugModel).Exec(ctx); slugInsertErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert slug: %w", slugInsertErr))
	}

	if publisher != nil {
		if err := publisher.PublishOrganizationCreated(ctx, events.OrganizationCreatedPayload{
			OrganizationID:    org.ID,
			OrganizationTitle: org.Title,
			TypeID:            org.OrganizationTypeID,
			ActorID:           localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if s.perms != nil && localUserID != 0 {
		orgID := fmt.Sprintf("%d", org.ID)
		if err := s.perms.LinkOrganizationToPlatform(ctx, orgID); err != nil {
			slog.WarnContext(ctx, "Failed to link organization to platform in SpiceDB", "error", err, "orgId", orgID)
		}
	}

	protoOrg := modelOrganizationToProto(org)
	protoOrg.Slug = orgSlug

	return connect.NewResponse(&eventsv1.CreateOrganizationResponse{
		Organization: protoOrg,
	}), nil
}
