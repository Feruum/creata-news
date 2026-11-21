package users

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/storage"
)

func (s *Service) GetUserAvatarUploadUrl(ctx context.Context, req *connect.Request[usersv1.GetUserAvatarUploadUrlRequest]) (*connect.Response[usersv1.GetUserAvatarUploadUrlResponse], error) {
	slog.Debug("GetUserAvatarUploadUrl", "filename", req.Msg.Filename, "contentType", req.Msg.ContentType)

	userID := auth.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("storage service is not available"))
	}

	objectKey := storage.GenerateUserAvatarKey(req.Msg.Filename)

	result, err := s.storage.GeneratePresignedUploadURL(ctx, objectKey, req.Msg.ContentType, 15*time.Minute)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to generate presigned URL", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate upload URL: %w", err))
	}

	return connect.NewResponse(&usersv1.GetUserAvatarUploadUrlResponse{
		UploadUrl: result.UploadURL,
		PublicUrl: result.PublicURL,
		ObjectKey: result.ObjectKey,
		Headers:   result.Headers,
	}), nil
}
