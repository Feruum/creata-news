package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"strconv"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/telegram"
)

// --- notification JSON types (matching existing telegram bot API) ---

type telegramNotification = telegram.Notification
type telegramBulkRequest = telegram.BulkRequest
type inlineKeyboardMarkup = telegram.ReplyMarkup
type inlineKeyboardButton = telegram.InlineKeyboardButton

func postBulkTelegramNotification(ctx context.Context, baseURL, secret string, body telegramBulkRequest) error {
	_, err := telegram.SendBulk(ctx, baseURL, secret, body)
	return err
}

// resolveOrgManagerTelegramIDs looks up org managers via SpiceDB and returns
// their Telegram IDs by querying the DB.
func resolveOrgManagerTelegramIDs(msg *message.Message, deps HandlerDeps, orgID int32) ([]int64, error) {
	if deps.Perms == nil || deps.DB == nil {
		return nil, nil
	}

	ctx := msg.Context()
	userIDStrs, err := deps.Perms.LookupSubjects(ctx, "organization", strconv.Itoa(int(orgID)), "manage_settings")
	if err != nil {
		return nil, fmt.Errorf("lookup organization managers: %w", err)
	}

	if len(userIDStrs) == 0 {
		return nil, nil
	}

	enforcer := authz.NewEnforcer(deps.DB, deps.Perms)
	var telegramIDs []int64
	for _, userIDStr := range userIDStrs {
		userID, err := strconv.ParseInt(userIDStr, 10, 32)
		if err != nil {
			continue
		}
		allowed, permissionErr := enforcer.CheckOrgPermissionWithError(ctx, int32(userID), orgID, "manage_settings")
		if permissionErr != nil {
			return nil, fmt.Errorf("check notification recipient: %w", permissionErr)
		}
		if !allowed {
			continue
		}
		user := &models.User{}
		err = deps.DB.NewSelect().Model(user).Where("id = ?", int32(userID)).Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("load notification recipient: %w", err)
		}
		if err != nil || user.TelegramID == nil {
			continue
		}
		telegramIDs = append(telegramIDs, *user.TelegramID)
	}

	return telegramIDs, nil
}

// --- telegram-notifier handlers ---

func handleCollaborationInvited(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.TelegramBotURL == "" {
			return nil
		}

		var payload events.CollaborationInvitedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal collaboration.invited: %w", err)
		}

		telegramIDs, err := resolveOrgManagerTelegramIDs(msg, deps, payload.OrganizationID)
		if err != nil {
			return err
		}
		notifications := make([]telegramNotification, 0, len(telegramIDs))
		for _, id := range telegramIDs {
			notifications = append(notifications, telegramNotification{TelegramID: id, Message: fmt.Sprintf("New collaboration invite for %s", payload.EventTitle)})
		}
		if len(notifications) == 0 {
			return nil
		}
		return postBulkTelegramNotification(msg.Context(), deps.TelegramBotURL, deps.InternalNotificationSecret, telegramBulkRequest{Notifications: notifications})
	}
}

func handleCollaborationAccepted(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.TelegramBotURL == "" {
			return nil
		}

		var payload events.CollaborationAcceptedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal collaboration.accepted: %w", err)
		}

		telegramIDs, err := resolveOrgManagerTelegramIDs(msg, deps, payload.HostOrganizationID)
		if err != nil {
			return err
		}
		if len(telegramIDs) == 0 {
			return nil
		}

		notifyMsg := fmt.Sprintf("✅ Organization has accepted the collaboration invite for event <b>%s</b>", payload.EventTitle)

		var notifications []telegramNotification
		for _, tid := range telegramIDs {
			notifications = append(notifications, telegramNotification{TelegramID: tid, Message: notifyMsg})
		}

		return postBulkTelegramNotification(msg.Context(), deps.TelegramBotURL, deps.InternalNotificationSecret, telegramBulkRequest{
			Notifications: notifications,
			ParseMode:     "HTML",
		})
	}
}

func handleCollaborationDeclined(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.TelegramBotURL == "" {
			return nil
		}

		var payload events.CollaborationDeclinedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal collaboration.declined: %w", err)
		}

		telegramIDs, err := resolveOrgManagerTelegramIDs(msg, deps, payload.HostOrganizationID)
		if err != nil {
			return err
		}
		if len(telegramIDs) == 0 {
			return nil
		}

		notifyMsg := fmt.Sprintf("❌ Organization has declined the collaboration invite for event <b>%s</b>", payload.EventTitle)

		var notifications []telegramNotification
		for _, tid := range telegramIDs {
			notifications = append(notifications, telegramNotification{TelegramID: tid, Message: notifyMsg})
		}

		return postBulkTelegramNotification(msg.Context(), deps.TelegramBotURL, deps.InternalNotificationSecret, telegramBulkRequest{
			Notifications: notifications,
			ParseMode:     "HTML",
		})
	}
}

func handleBookingApproved(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.TelegramBotURL == "" {
			return nil
		}

		var payload events.BookingApprovedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal booking.approved: %w", err)
		}

		telegramIDs, err := resolveOrgManagerTelegramIDs(msg, deps, payload.OrganizationID)
		if err != nil {
			return err
		}
		if len(telegramIDs) == 0 {
			return nil
		}

		notifyMsg := fmt.Sprintf("✅ <b>Room booking approved</b>\n\nYour booking request <b>%s</b> has been approved.", payload.EventTitle)

		var notifications []telegramNotification
		for _, tid := range telegramIDs {
			notifications = append(notifications, telegramNotification{TelegramID: tid, Message: notifyMsg})
		}

		return postBulkTelegramNotification(msg.Context(), deps.TelegramBotURL, deps.InternalNotificationSecret, telegramBulkRequest{
			Notifications: notifications,
			ParseMode:     "HTML",
		})
	}
}

func handleBookingRejected(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.TelegramBotURL == "" {
			return nil
		}

		var payload events.BookingRejectedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal booking.rejected: %w", err)
		}

		telegramIDs, err := resolveOrgManagerTelegramIDs(msg, deps, payload.OrganizationID)
		if err != nil {
			return err
		}
		if len(telegramIDs) == 0 {
			return nil
		}

		notifyMsg := fmt.Sprintf("❌ <b>Room booking rejected</b>\n\nYour booking request <b>%s</b> was rejected.", payload.EventTitle)

		var notifications []telegramNotification
		for _, tid := range telegramIDs {
			notifications = append(notifications, telegramNotification{TelegramID: tid, Message: notifyMsg})
		}

		return postBulkTelegramNotification(msg.Context(), deps.TelegramBotURL, deps.InternalNotificationSecret, telegramBulkRequest{
			Notifications: notifications,
			ParseMode:     "HTML",
		})
	}
}

func handleEventPublished(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.TelegramBotURL == "" {
			return nil
		}

		var payload events.EventPublishedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal event.published: %w", err)
		}

		if deps.DB == nil {
			return nil
		}

		ctx := msg.Context()

		// Outbox delivery can lag behind an edit, unpublish, or deletion.
		visible, err := deps.DB.NewSelect().Model((*models.Event)(nil)).Column("id").
			Where("id = ?", payload.EventID).
			Where("status = ?", models.EventStatusPublished).
			Where("publish_on_map = TRUE").Exists(ctx)
		if err != nil {
			return fmt.Errorf("check announcement publication: %w", err)
		}
		if !visible {
			return nil
		}

		var subscribers []struct {
			TelegramID int64 `bun:"telegram_id"`
		}
		err = deps.DB.NewSelect().
			TableExpr("organization_subscriptions AS os").
			Join("JOIN users AS u ON u.id = os.user_id").
			ColumnExpr("u.telegram_id").
			Where("os.organization_id = ?", payload.OrganizationID).
			Where("u.telegram_id IS NOT NULL").
			Scan(ctx, &subscribers)
		if err != nil {
			return fmt.Errorf("load notification subscribers: %w", err)
		}

		if len(subscribers) == 0 {
			return nil
		}

		eventURL := fmt.Sprintf("%s/event/%s", payload.MapAppURL, payload.EventSlug)

		notifyMsg := fmt.Sprintf(
			"📢 <b>%s</b> just published a new event!\n\n<b>%s</b>\n📅 %s",
			payload.OrganizationName,
			payload.Title,
			payload.StartTime,
		)
		if payload.Location != "" {
			notifyMsg += fmt.Sprintf("\n📍 %s", payload.Location)
		}
		notifyMsg += fmt.Sprintf("\n\n%s", eventURL)

		var notifications []telegramNotification
		for _, sub := range subscribers {
			notifications = append(notifications, telegramNotification{
				TelegramID: sub.TelegramID,
				Message:    notifyMsg,
			})
		}

		return postBulkTelegramNotification(msg.Context(), deps.TelegramBotURL, deps.InternalNotificationSecret, telegramBulkRequest{
			Notifications: notifications,
			ParseMode:     "HTML",
			ReplyMarkup: &inlineKeyboardMarkup{
				InlineKeyboard: [][]inlineKeyboardButton{
					{{Text: "View Event →", URL: eventURL}},
				},
			},
		})
	}
}
