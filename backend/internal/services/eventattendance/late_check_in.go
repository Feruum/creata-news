package eventattendance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) LateCheckIn(ctx context.Context, req *connect.Request[eventsv1.LateCheckInRequest]) (*connect.Response[eventsv1.LateCheckInResponse], error) {
	return attendanceMutation(ctx, s, req.Msg.RegistrationId, false, func(ctx context.Context, db bun.IDB, actor int32) (*connect.Response[eventsv1.LateCheckInResponse], error) {
		return s.lateCheckIn(ctx, db, req, actor)
	})
}
func (s *Service) lateCheckIn(ctx context.Context, db bun.IDB, req *connect.Request[eventsv1.LateCheckInRequest], localUserID int32) (*connect.Response[eventsv1.LateCheckInResponse], error) {
	slog.Debug("LateCheckIn", "registrationId", req.Msg.RegistrationId)

	if req.Msg.Reason == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("reason is required for late check-in"))
	}

	// Fetch registration
	var reg models.EventRegistration
	err := db.NewSelect().Model(&reg).Where("id = ?", req.Msg.RegistrationId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("registration not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if reg.Status != models.RegistrationStatusRegistered {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("only confirmed registrations may be checked in"))
	}

	// Fetch event and verify within 7-day window
	var event models.Event
	err = db.NewSelect().Model(&event).Where("id = ?", reg.EventID).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	sevenDaysAfterEnd := event.EndTime.Add(7 * 24 * time.Hour)
	if time.Now().After(sevenDaysAfterEnd) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("late check-in window has expired (7 days after event end)"))
	}

	// Upsert attendance record
	method := models.CheckInMethodLate
	reason := req.Msg.Reason
	var notes *string
	if req.Msg.Notes != nil {
		notes = req.Msg.Notes
	}

	var existingAtt models.EventAttendance
	err = db.NewSelect().Model(&existingAtt).Where("registration_id = ?", req.Msg.RegistrationId).Scan(ctx)

	var att models.EventAttendance
	if err == nil {
		// Update existing
		att = existingAtt
		att.Status = models.AttendanceStatusCheckedIn
		att.CheckedInAt = timePtr(time.Now())
		att.CheckedInBy = &localUserID
		att.Method = &method
		att.Reason = &reason
		att.Notes = notes
		att.UpdatedAt = time.Now()
		_, err = db.NewUpdate().
			Model(&att).
			Column("status", "checked_in_at", "checked_in_by", "method", "reason", "notes", "updated_at").
			Where("id = ?", att.ID).
			Returning("*").
			Exec(ctx)
	} else if errors.Is(err, sql.ErrNoRows) {
		att = models.EventAttendance{
			RegistrationID: req.Msg.RegistrationId,
			Status:         models.AttendanceStatusCheckedIn,
			CheckedInAt:    timePtr(time.Now()),
			CheckedInBy:    &localUserID,
			Method:         &method,
			Reason:         &reason,
			Notes:          notes,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err = db.NewInsert().Model(&att).Returning("*").Exec(ctx)
	} else {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.LateCheckInResponse{
		Attendance: dbEventAttendanceToProto(att),
	}), nil
}
