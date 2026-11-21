package organizations

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

// Direct upload credentials bypass byte validation and are no longer issued.
func (s *Service) GetOrganizationImageUploadUrl(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationImageUploadUrlRequest]) (*connect.Response[eventsv1.GetOrganizationImageUploadUrlResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("use UploadOrganizationImage to upload a validated image"))
}
