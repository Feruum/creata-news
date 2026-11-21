package organizations

import (
	"connectrpc.com/connect"
	"context"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/imageupload"
)

func (s *Service) UploadOrganizationImage(ctx context.Context, req *connect.Request[eventsv1.UploadOrganizationImageRequest]) (*connect.Response[eventsv1.UploadOrganizationImageResponse], error) {
	uploader := imageupload.Service{DB: s.db, Authz: s.authz, Storage: s.storage}
	publicURL, key, err := uploader.UploadOrganization(ctx, req.Msg.Image, req.Msg.ContentType, req.Msg.OrganizationId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&eventsv1.UploadOrganizationImageResponse{PublicUrl: publicURL, ObjectKey: key}), nil
}
