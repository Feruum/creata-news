package events

import (
	"connectrpc.com/connect"
	"context"
	"github.com/google/uuid"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/imageupload"
)

func (s *Service) UploadEventImage(ctx context.Context, req *connect.Request[eventsv1.UploadEventImageRequest]) (*connect.Response[eventsv1.UploadEventImageResponse], error) {
	uploader := imageupload.Service{DB: s.db, Authz: s.authz, Storage: s.storage}
	publicURL, key, err := uploader.UploadEvent(ctx, req.Msg.Image, req.Msg.ContentType, req.Msg.EventId, req.Msg.OrganizationId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&eventsv1.UploadEventImageResponse{PublicUrl: publicURL, ObjectKey: key}), nil
}

func (s *Service) UploadDraftCover(ctx context.Context, image []byte, mime string, orgID int32, objectID uuid.UUID) (string, error) {
	uploader := imageupload.Service{DB: s.db, Authz: s.authz, Storage: s.storage}
	publicURL, _, err := uploader.UploadDraftCover(ctx, image, mime, orgID, objectID)
	return publicURL, err
}
