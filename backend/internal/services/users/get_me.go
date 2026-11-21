package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetMe(ctx context.Context, req *connect.Request[usersv1.GetMeRequest]) (*connect.Response[usersv1.GetMeResponse], error) {
	kratosUserID := auth.GetUserID(ctx)
	if kratosUserID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	slog.Debug("GetMe", "kratosUserID", kratosUserID)

	var user models.User
	err := s.db.NewSelect().Model(&user).
		Column("id", "kratos_id", "username", "email", "password", "avatar_url", "created_at", "updated_at", "first_name", "last_name", "telegram_id", "telegram_username", "telegram_linked_at").
		Where("kratos_id = ?", kratosUserID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			user, err = s.syncUserFromKratos(ctx, kratosUserID)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to sync user from Kratos", "error", err, "kratosId", kratosUserID)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sync user: %w", err))
			}
		} else {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	protoUser := s.modelUserToProto(ctx, user)

	// modelUserToProto has already checked current grants through the enforcer.
	// Use the same verified snapshot for both role and permissions in this response.
	permissions := &usersv1.UserPermissions{
		IsAdmin:        protoUser.PlatformRole == usersv1.PlatformRole_PLATFORM_ROLE_ADMIN,
		IsStaff:        protoUser.PlatformRole == usersv1.PlatformRole_PLATFORM_ROLE_ADMIN || protoUser.PlatformRole == usersv1.PlatformRole_PLATFORM_ROLE_STAFF,
		ManagedClubIds: []int32{},
	}
	if s.perms != nil {
		managedOrgs, err := s.perms.GetManagedOrganizations(ctx, fmt.Sprintf("%d", user.ID))
		if err != nil {
			slog.WarnContext(ctx, "Failed to fetch managed organizations from SpiceDB", "error", err, "userId", user.ID)
		} else {
			permissions.ManagedClubIds = s.authz.FilterOrganizations(ctx, user.ID, managedOrgs, "manage_settings")
		}
	}

	return connect.NewResponse(&usersv1.GetMeResponse{
		User:        protoUser,
		Permissions: permissions,
	}), nil
}
