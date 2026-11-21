package users

import (
	"context"
	"github.com/uptrace/bun"
	"strings"
	"time"

	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) modelUserToProto(ctx context.Context, u models.User) *usersv1.User {
	protoUser := &usersv1.User{
		Id:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		AvatarUrl:    u.AvatarURL,
		CreatedAt:    u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    u.UpdatedAt.Format(time.RFC3339),
		PlatformRole: usersv1.PlatformRole_PLATFORM_ROLE_USER,
	}

	if u.FirstName != nil {
		protoUser.FirstName = u.FirstName
	}
	if u.LastName != nil {
		protoUser.LastName = u.LastName
	}

	if u.TelegramID != nil {
		telegramID := *u.TelegramID
		protoUser.TelegramId = &telegramID
	}
	if u.TelegramUsername != nil {
		protoUser.TelegramUsername = u.TelegramUsername
	}
	if u.TelegramLinkedAt != nil {
		linkedAt := u.TelegramLinkedAt.Format(time.RFC3339)
		protoUser.TelegramLinkedAt = &linkedAt
	}

	if s.authz != nil {
		if s.authz.CheckPlatformAdmin(ctx, u.ID) {
			protoUser.PlatformRole = usersv1.PlatformRole_PLATFORM_ROLE_ADMIN
			return protoUser
		}

		if s.authz.CheckPlatformStaff(ctx, u.ID) {
			protoUser.PlatformRole = usersv1.PlatformRole_PLATFORM_ROLE_STAFF
			return protoUser
		}
	}

	return protoUser
}

func modelUserToPublicProto(u models.User) *usersv1.User {
	protoUser := &usersv1.User{
		Id:        u.ID,
		Username:  u.Username,
		AvatarUrl: u.AvatarURL,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
	if u.FirstName != nil {
		protoUser.FirstName = u.FirstName
	}
	if u.LastName != nil {
		protoUser.LastName = u.LastName
	}
	return protoUser
}

func modelPreRegToProto(p models.PreRegisteredUser) *usersv1.PreRegisteredUser {
	proto := &usersv1.PreRegisteredUser{
		Id:        p.ID,
		Email:     p.Email,
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
	}

	switch p.PlatformRole {
	case models.PlatformRoleAdmin:
		proto.PlatformRole = usersv1.PlatformRole_PLATFORM_ROLE_ADMIN
	case models.PlatformRoleStaff:
		proto.PlatformRole = usersv1.PlatformRole_PLATFORM_ROLE_STAFF
	}

	if p.CreatedBy != nil {
		createdBy := *p.CreatedBy
		proto.CreatedBy = &createdBy
	}
	if p.UsedAt != nil {
		usedAt := p.UsedAt.Format(time.RFC3339)
		proto.UsedAt = &usedAt
	}
	if p.UsedByUserID != nil {
		usedByUserID := *p.UsedByUserID
		proto.UsedByUserId = &usedByUserID
	}

	return proto
}

func (s *Service) syncUserFromKratos(ctx context.Context, kratosUserID string) (models.User, error) {
	email := auth.GetUserEmail(ctx)
	if email == "" {
		email = kratosUserID + "@placeholder.local"
	}

	username := email
	if atIdx := strings.Index(email, "@"); atIdx > 0 {
		username = email[:atIdx]
	}

	firstName := auth.GetUserFirstName(ctx)
	lastName := auth.GetUserLastName(ctx)

	var firstNameValue *string
	if firstName != "" {
		firstNameValue = &firstName
	}
	var lastNameValue *string
	if lastName != "" {
		lastNameValue = &lastName
	}

	var user models.User
	var roleChanged bool
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewRaw(`
		INSERT INTO users (kratos_id, username, email, password, first_name, last_name)
		VALUES (?, ?, ?, 'kratos-managed', ?, ?)
		ON CONFLICT (email) DO UPDATE SET
			kratos_id = ?,
			first_name = COALESCE(?, users.first_name),
			last_name = COALESCE(?, users.last_name),
			updated_at = NOW()
		RETURNING *
	`, kratosUserID, username, email, firstNameValue, lastNameValue, kratosUserID, firstNameValue, lastNameValue).Scan(ctx, &user)
		if err != nil {
			return err
		}
		roleChanged, err = s.applyPreRegisteredRoleInTx(ctx, tx, email, user.ID)
		return err
	})
	if err != nil {
		return models.User{}, err
	}

	if roleChanged {
		if err := s.authz.SyncPlatformUserRole(ctx, user.ID); err != nil {
			return models.User{}, err
		}
	}

	return user, nil
}
