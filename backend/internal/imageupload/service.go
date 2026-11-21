// Package imageupload owns authorization and validation for public raster uploads.
package imageupload

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/uptrace/bun"
)

type Service struct {
	DB      bun.IDB
	Authz   *authz.Enforcer
	Storage storage.Provider
}

func (s Service) UploadEvent(ctx context.Context, image []byte, mime string, eventID, orgID int32) (string, string, error) {
	return s.uploadEvent(ctx, image, mime, eventID, orgID, uuid.New())
}

// UploadDraftCover accepts a server-derived object ID so a retried transaction
// rewrites the same validated object after a storage success and database rollback.
func (s Service) UploadDraftCover(ctx context.Context, image []byte, mime string, orgID int32, objectID uuid.UUID) (string, string, error) {
	return s.uploadEvent(ctx, image, mime, 0, orgID, objectID)
}

func (s Service) uploadEvent(ctx context.Context, image []byte, mime string, eventID, orgID int32, objectID uuid.UUID) (string, string, error) {
	uid, err := s.localUser(ctx)
	if err != nil {
		return "", "", err
	}
	if eventID < 0 || orgID < 0 || (eventID == 0) == (orgID == 0) {
		return "", "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provide either event_id or organization_id"))
	}
	scope := ""
	if eventID > 0 {
		var event models.Event
		if err := s.DB.NewSelect().Model(&event).Where("id = ?", eventID).Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", "", connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
			}
			return "", "", connect.NewError(connect.CodeInternal, fmt.Errorf("load event: %w", err))
		}
		if _, err := s.Authz.RequireResourcePermission(ctx, "event", fmt.Sprint(eventID), "edit"); err != nil {
			return "", "", err
		}
		scope = fmt.Sprintf("events/%d", eventID)
	} else {
		if err := s.requireOrganization(ctx, orgID); err != nil {
			return "", "", err
		}
		if _, err := s.Authz.RequireOrgPermission(ctx, orgID, "create_event"); err != nil {
			return "", "", err
		}
		scope = fmt.Sprintf("events/org-%d", orgID)
	}
	return s.store(ctx, scope, uid, image, mime, objectID)
}
func (s Service) UploadOrganization(ctx context.Context, image []byte, mime string, orgID int32) (string, string, error) {
	uid, err := s.localUser(ctx)
	if err != nil {
		return "", "", err
	}
	if orgID < 0 {
		return "", "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id must not be negative"))
	}
	scope := "organizations/drafts"
	if orgID == 0 {
		if _, err := s.Authz.RequirePlatformStaff(ctx); err != nil {
			return "", "", err
		}
	} else {
		if err := s.requireOrganization(ctx, orgID); err != nil {
			return "", "", err
		}
		if _, err := s.Authz.RequireOrgPermission(ctx, orgID, "manage_settings"); err != nil {
			return "", "", err
		}
		scope = fmt.Sprintf("organizations/%d", orgID)
	}
	return s.store(ctx, scope, uid, image, mime, uuid.New())
}
func (s Service) localUser(ctx context.Context) (int32, error) {
	if s.Authz == nil {
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("authorization unavailable"))
	}
	return s.Authz.GetLocalUserID(ctx)
}
func (s Service) requireOrganization(ctx context.Context, id int32) error {
	exists, err := s.DB.NewSelect().Model((*models.Organization)(nil)).Where("id = ?", id).Exists(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("load organization: %w", err))
	}
	if !exists {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}
	return nil
}
func (s Service) store(ctx context.Context, scope string, userID int32, image []byte, mime string, objectID uuid.UUID) (string, string, error) {
	if s.Storage == nil {
		return "", "", connect.NewError(connect.CodeUnavailable, fmt.Errorf("storage unavailable"))
	}
	data, err := storage.NormalizeImage(ctx, image, mime)
	if err != nil {
		return "", "", connect.NewError(connect.CodeInvalidArgument, err)
	}
	key := fmt.Sprintf("%s/u%d/%s.png", scope, userID, objectID)
	if err := s.Storage.UploadObject(ctx, key, "image/png", data); err != nil {
		return "", "", connect.NewError(connect.CodeInternal, fmt.Errorf("upload image: %w", err))
	}
	return s.Storage.GetPublicURL(key), key, nil
}
