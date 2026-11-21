package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/util"
)

type pendingReminderRow struct {
	EventID            int32           `bun:"event_id"`
	EventTitle         string          `bun:"event_title"`
	EventDescription   json.RawMessage `bun:"event_description"`
	StartTime          time.Time       `bun:"start_time"`
	EndTime            time.Time       `bun:"end_time"`
	StructuredLocation json.RawMessage `bun:"structured_location"`
	UserID             int32           `bun:"user_id"`
	TelegramID         int64           `bun:"telegram_id"`
	UserFirstName      *string         `bun:"first_name"`
	OrganizationTitle  string          `bun:"organization_title"`
	ImageUrl           *string         `bun:"image_url"`
	Slug               string          `bun:"slug"`
}

func (s *Service) tableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	err := s.db.NewRaw(`SELECT to_regclass(?) IS NOT NULL`, tableName).Scan(ctx, &exists)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, err)
	}
	return exists, nil
}

func (s *Service) eventsColumnExists(ctx context.Context, columnName string) (bool, error) {
	var exists bool
	err := s.db.NewRaw(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'events'
			  AND column_name = ?
		)
	`, columnName).Scan(ctx, &exists)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, err)
	}
	return exists, nil
}

func (s *Service) GetPendingReminders(ctx context.Context, req *connect.Request[eventsv1.GetPendingRemindersRequest]) (*connect.Response[eventsv1.GetPendingRemindersResponse], error) {
	if err := auth.RequireInternalSecret(req, s.internalReminderSecret); err != nil {
		return nil, err
	}

	var windowStart, windowEnd time.Duration
	switch req.Msg.ReminderType {
	case "day_before":
		windowStart = 23 * time.Hour
		windowEnd = 25 * time.Hour
	case "hour_before":
		windowStart = 55 * time.Minute
		windowEnd = 65 * time.Minute
	case "15_min":
		windowStart = 10 * time.Minute
		windowEnd = 20 * time.Minute
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid reminder_type: %s", req.Msg.ReminderType))
	}

	now := time.Now()
	fromTime := now.Add(windowStart)
	toTime := now.Add(windowEnd)

	hasEventStatus, err := s.eventsColumnExists(ctx, "status")
	if err != nil {
		return nil, err
	}
	hasEventRemindersTable, err := s.tableExists(ctx, "event_reminders")
	if err != nil {
		return nil, err
	}

	statusClause := ""
	if hasEventStatus {
		statusClause = "AND e.status = 'published'"
	} else {
		slog.Warn("events.status column is missing; skipping published-only filter")
	}

	reminderDedupClause := ""
	if hasEventRemindersTable {
		reminderDedupClause = `
		  AND NOT EXISTS (
			  SELECT 1 FROM event_reminders evr
			  WHERE evr.event_id = e.id AND evr.user_id = u.id AND evr.reminder_type = ?
		  )`
	} else {
		slog.Warn("event_reminders table is missing; reminder deduplication is disabled")
	}

	query := fmt.Sprintf(`
		SELECT
			e.id AS event_id,
			e.title AS event_title,
			e.description AS event_description,
			e.start_time,
			e.end_time,
			e.structured_location,
			u.id AS user_id,
			u.telegram_id,
			u.first_name,
			o.title AS organization_title,
			e.image_url,
			es.slug
		FROM events e
		JOIN event_registrations r ON r.event_id = e.id AND r.status = 'registered'
		JOIN users u ON u.id = r.user_id AND u.telegram_id IS NOT NULL
		JOIN organizations o ON o.id = e.organization_id
		LEFT JOIN event_slugs es ON es.event_id = e.id AND es.is_current = true
		WHERE e.start_time >= ?
		  AND e.start_time <= ?
		  %s
		  %s
		ORDER BY e.start_time, u.id
	`, statusClause, reminderDedupClause)

	params := []any{fromTime, toTime}
	if hasEventRemindersTable {
		params = append(params, req.Msg.ReminderType)
	}

	var rows []pendingReminderRow
	err = s.db.NewRaw(query, params...).Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	reminders := make([]*eventsv1.PendingReminder, len(rows))
	for i, row := range rows {
		userFirstName := ""
		if row.UserFirstName != nil {
			userFirstName = *row.UserFirstName
		}
		reminder := &eventsv1.PendingReminder{
			EventId:           row.EventID,
			EventTitle:        row.EventTitle,
			EventDescription:  util.ExtractPlainText(row.EventDescription),
			StartTime:         row.StartTime.Format(time.RFC3339),
			EndTime:           row.EndTime.Format(time.RFC3339),
			UserId:            row.UserID,
			TelegramId:        row.TelegramID,
			UserFirstName:     userFirstName,
			ReminderType:      req.Msg.ReminderType,
			OrganizationTitle: row.OrganizationTitle,
			Slug:              row.Slug,
		}
		if row.ImageUrl != nil {
			reminder.EventImageUrl = row.ImageUrl
		}
		if len(row.StructuredLocation) > 0 {
			reminder.Location = parseStructuredLocation(row.StructuredLocation)
		}
		reminders[i] = reminder
	}

	slog.Debug("GetPendingReminders", "type", req.Msg.ReminderType, "count", len(reminders))
	return connect.NewResponse(&eventsv1.GetPendingRemindersResponse{Reminders: reminders}), nil
}

func (s *Service) MarkRemindersSent(ctx context.Context, req *connect.Request[eventsv1.MarkRemindersSentRequest]) (*connect.Response[eventsv1.MarkRemindersSentResponse], error) {
	if err := auth.RequireInternalSecret(req, s.internalReminderSecret); err != nil {
		return nil, err
	}

	if len(req.Msg.Reminders) == 0 {
		return connect.NewResponse(&eventsv1.MarkRemindersSentResponse{Marked: 0}), nil
	}

	hasEventRemindersTable, err := s.tableExists(ctx, "event_reminders")
	if err != nil {
		return nil, err
	}
	if !hasEventRemindersTable {
		slog.Warn("event_reminders table is missing; skipping reminder persistence", "requested", len(req.Msg.Reminders))
		return connect.NewResponse(&eventsv1.MarkRemindersSentResponse{Marked: 0}), nil
	}

	reminders := make([]models.EventReminder, len(req.Msg.Reminders))
	for i, reminder := range req.Msg.Reminders {
		reminders[i] = models.EventReminder{
			EventID:      reminder.EventId,
			UserID:       reminder.UserId,
			ReminderType: models.ReminderType(reminder.ReminderType),
		}
		if reminder.TelegramMessageId != nil {
			reminders[i].TelegramMessageID = reminder.TelegramMessageId
		}
	}

	result, err := s.db.NewInsert().Model(&reminders).
		On("CONFLICT (event_id, user_id, reminder_type) DO NOTHING").
		Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to mark reminders sent", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	affected, _ := result.RowsAffected()

	slog.Debug("MarkRemindersSent", "requested", len(req.Msg.Reminders), "inserted", affected)
	return connect.NewResponse(&eventsv1.MarkRemindersSentResponse{Marked: int32(affected)}), nil
}
