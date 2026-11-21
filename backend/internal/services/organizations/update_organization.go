package organizations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/services/orgmembers"
	"github.com/studyverse/ems-backend/internal/slug"
)

func (s *Service) UpdateOrganization(ctx context.Context, req *connect.Request[eventsv1.UpdateOrganizationRequest]) (*connect.Response[eventsv1.UpdateOrganizationResponse], error) {
	slog.Debug("UpdateOrganization", "id", req.Msg.Id)

	_, err := s.authz.RequireOrgPermission(ctx, req.Msg.Id, "manage_settings")
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	// Capture old state for archive detection
	var oldOrg models.Organization
	if err := tx.NewSelect().Model(&oldOrg).Where("id = ?", req.Msg.Id).Scan(ctx); err != nil {
		slog.Warn("Failed to fetch old org state", "error", err)
	}

	query := tx.NewUpdate().Model((*models.Organization)(nil)).Where("id = ?", req.Msg.Id)

	if req.Msg.Title != nil {
		query = query.Set("title = ?", *req.Msg.Title)
	}
	if req.Msg.ImageUrl != nil {
		query = query.Set("image_url = ?", *req.Msg.ImageUrl)
	}
	if req.Msg.Description != nil {
		query = query.Set("description = ?", json.RawMessage(*req.Msg.Description))
	}
	if req.Msg.OrganizationTypeId != nil {
		query = query.Set("organization_type_id = ?", *req.Msg.OrganizationTypeId)
	}
	if req.Msg.Instagram != nil {
		query = query.Set("instagram = ?", *req.Msg.Instagram)
	}
	if req.Msg.TelegramChannel != nil {
		query = query.Set("telegram_channel = ?", *req.Msg.TelegramChannel)
	}
	if req.Msg.TelegramChat != nil {
		query = query.Set("telegram_chat = ?", *req.Msg.TelegramChat)
	}
	if req.Msg.Website != nil {
		query = query.Set("website = ?", *req.Msg.Website)
	}
	if req.Msg.Youtube != nil {
		query = query.Set("youtube = ?", *req.Msg.Youtube)
	}
	if req.Msg.Tiktok != nil {
		query = query.Set("tiktok = ?", *req.Msg.Tiktok)
	}
	if req.Msg.Linkedin != nil {
		query = query.Set("linkedin = ?", *req.Msg.Linkedin)
	}
	if req.Msg.Status != nil {
		status := protoStatusToDB(*req.Msg.Status)
		query = query.Set("status = ?", status)
	}

	var org models.Organization
	err = query.Returning("*").Scan(ctx, &org)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if req.Msg.Status != nil {
		newStatus := protoStatusToDB(*req.Msg.Status)
		oldStatus := models.OrganizationStatus("")
		if oldOrg.Status != nil {
			oldStatus = *oldOrg.Status
		}
		if newStatus == models.OrganizationStatusArchived && oldStatus != models.OrganizationStatusArchived {
			if s.orgMembers != nil {
				localUserID, _ := s.authz.GetLocalUserID(ctx)
				if err := orgmembers.RecordBulkMembershipHistoryInTransaction(ctx, tx, org.ID, org.Title, models.MembershipHistoryActionOrgArchived, &localUserID); err != nil {
					return nil, connect.NewError(connect.CodeInternal, err)
				}
			}
		}
	}

	if req.Msg.Title != nil {
		baseSlug := slug.Slugify(*req.Msg.Title)
		orgSlug := baseSlug
		for i := 1; i <= 100; i++ {
			exists, checkErr := tx.NewSelect().Model((*models.OrganizationSlug)(nil)).Where("slug = ?", orgSlug).Exists(ctx)
			if checkErr != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check slug: %w", checkErr))
			}
			if !exists {
				break
			}
			orgSlug = slug.WithSuffix(baseSlug, i+1)
		}
		if _, err := tx.NewUpdate().Model((*models.OrganizationSlug)(nil)).Set("is_current = ?", false).Where("organization_id = ?", org.ID).Exec(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to mark old slugs: %w", err))
		}
		newSlug := models.OrganizationSlug{
			OrganizationID: org.ID,
			Slug:           orgSlug,
			IsCurrent:      true,
		}
		if _, err := tx.NewInsert().Model(&newSlug).Exec(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert new slug: %w", err))
		}
	}

	localUserID, _ := s.authz.GetLocalUserID(ctx)
	isArchived := org.Status != nil && *org.Status == models.OrganizationStatusArchived
	if publisher != nil {
		if err := publisher.PublishOrganizationUpdated(ctx, events.OrganizationUpdatedPayload{
			OrganizationID:    org.ID,
			OrganizationTitle: org.Title,
			TypeID:            org.OrganizationTypeID,
			IsArchived:        isArchived,
			ActorID:           localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	protoOrg := modelOrganizationToProto(org)
	var orgSlug models.OrganizationSlug
	slugErr := tx.NewSelect().
		Model(&orgSlug).
		Column("slug").
		Where("organization_id = ?", org.ID).
		Where("is_current = true").
		Limit(1).
		Scan(ctx)
	if slugErr == nil {
		protoOrg.Slug = orgSlug.Slug
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.UpdateOrganizationResponse{
		Organization: protoOrg,
	}), nil
}
