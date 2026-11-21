package users

import (
	"context"
	"database/sql"

	"errors"
	"fmt"
	"log/slog"

	"time"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/telegram"
)

func (s *Service) LinkTelegram(ctx context.Context, req *connect.Request[usersv1.LinkTelegramRequest]) (*connect.Response[usersv1.LinkTelegramResponse], error) {
	kratosUserID := auth.GetUserID(ctx)
	if kratosUserID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	loginData := &telegram.LoginData{
		ID:        req.Msg.TelegramId,
		FirstName: req.Msg.FirstName,
		LastName:  req.Msg.LastName,
		Username:  req.Msg.TelegramUsername,
		PhotoURL:  req.Msg.PhotoUrl,
		AuthDate:  req.Msg.AuthDate,
		Hash:      req.Msg.Hash,
	}

	if err := telegram.VerifyLogin(loginData, s.telegramBotToken, 24*time.Hour); err != nil {
		slog.WarnContext(ctx, "Telegram login verification failed", "error", err, "telegramId", req.Msg.TelegramId)
		if errors.Is(err, telegram.ErrMissingToken) {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("telegram verification not configured"))
		}
		if errors.Is(err, telegram.ErrExpiredAuth) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("telegram login has expired, please try again"))
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid telegram authentication"))
	}

	var user models.User
	err := s.db.NewSelect().Model(&user).Where("kratos_id = ?", kratosUserID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var existingUser models.User
	err = s.db.NewSelect().Model(&existingUser).Where("telegram_id = ?", req.Msg.TelegramId).Scan(ctx)
	if err == nil && existingUser.ID != user.ID {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("telegram account already linked to another user"))
	}

	var telegramUsername *string
	if req.Msg.TelegramUsername != "" {
		telegramUsername = &req.Msg.TelegramUsername
	}
	var updatedUser models.User
	err = s.db.NewUpdate().
		Model(&updatedUser).
		Set("telegram_id = ?", req.Msg.TelegramId).
		Set("telegram_username = ?", telegramUsername).
		Set("avatar_url = COALESCE(NULLIF(avatar_url, ''), NULLIF(?, ''))", loginData.PhotoURL).
		Set("telegram_linked_at = NOW()").
		Set("updated_at = NOW()").
		Where("id = ?", user.ID).
		Returning("*").
		Scan(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to link telegram account", "error", err, "userId", user.ID)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to link telegram: %w", err))
	}

	slog.InfoContext(ctx, "Linked Telegram account", "userId", user.ID, "telegramId", req.Msg.TelegramId, "username", req.Msg.TelegramUsername)

	return connect.NewResponse(&usersv1.LinkTelegramResponse{
		User: s.modelUserToProto(ctx, updatedUser),
	}), nil
}

func (s *Service) UnlinkTelegram(ctx context.Context, req *connect.Request[usersv1.UnlinkTelegramRequest]) (*connect.Response[usersv1.UnlinkTelegramResponse], error) {
	kratosUserID := auth.GetUserID(ctx)
	if kratosUserID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}

	var user models.User
	err := s.db.NewSelect().Model(&user).Where("kratos_id = ?", kratosUserID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var updatedUser models.User
	err = s.db.NewUpdate().
		Model(&updatedUser).
		Set("telegram_id = NULL").
		Set("telegram_username = NULL").
		Set("telegram_linked_at = NULL").
		Set("updated_at = NOW()").
		Where("id = ?", user.ID).
		Returning("*").
		Scan(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to unlink telegram account", "error", err, "userId", user.ID)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unlink telegram: %w", err))
	}

	slog.InfoContext(ctx, "Unlinked Telegram account", "userId", user.ID)

	return connect.NewResponse(&usersv1.UnlinkTelegramResponse{
		User: s.modelUserToProto(ctx, updatedUser),
	}), nil
}

func (s *Service) SendTelegramNotification(ctx context.Context, req *connect.Request[usersv1.SendTelegramNotificationRequest]) (*connect.Response[usersv1.SendTelegramNotificationResponse], error) {
	if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
		return nil, err
	}
	if !telegram.ValidMessage(req.Msg.Message) || !telegram.ValidParseMode(req.Msg.GetParseMode()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("message must contain 1-4096 characters and a supported parse mode"))
	}
	if s.telegramBotURL == "" || s.internalNotificationSecret == "" {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("internal telegram notification service is not configured"))
	}
	notifications := make([]telegram.Notification, 0, len(req.Msg.UserIds))
	var skipped int32
	for _, userID := range req.Msg.UserIds {
		var user models.User
		err := s.db.NewSelect().Model(&user).Where("id=?", userID).Scan(ctx)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load notification recipient: %w", err))
			}
			skipped++
			continue
		}
		if user.TelegramID == nil {
			skipped++
			continue
		}
		notifications = append(notifications, telegram.Notification{TelegramID: *user.TelegramID, Message: req.Msg.Message})
	}
	result, err := telegram.SendBulk(ctx, s.telegramBotURL, s.internalNotificationSecret, telegram.BulkRequest{Notifications: notifications, ParseMode: req.Msg.GetParseMode()})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("send telegram notifications: %w", err))
	}
	return connect.NewResponse(&usersv1.SendTelegramNotificationResponse{Sent: result.Sent, Failed: result.Failed, Skipped: skipped}), nil
}
