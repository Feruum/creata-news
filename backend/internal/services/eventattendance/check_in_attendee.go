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

func (s *Service) CheckInAttendee(ctx context.Context, req *connect.Request[eventsv1.CheckInAttendeeRequest]) (*connect.Response[eventsv1.CheckInAttendeeResponse], error) {
	return attendanceMutation(ctx, s, req.Msg.RegistrationId, false, func(ctx context.Context, db bun.IDB, actor int32) (*connect.Response[eventsv1.CheckInAttendeeResponse], error) {
		return s.checkInAttendee(ctx, db, req)
	})
}
func (s *Service) checkInAttendee(ctx context.Context, db bun.IDB, req *connect.Request[eventsv1.CheckInAttendeeRequest]) (*connect.Response[eventsv1.CheckInAttendeeResponse], error) {
	slog.Debug("CheckInAttendee", "registrationId", req.Msg.RegistrationId)

	// Check if registration exists
	var reg models.EventRegistration
	err := db.NewSelect().Model(&reg).Where("id = ?", req.Msg.RegistrationId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if reg.Status != models.RegistrationStatusRegistered {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("registration status %q cannot be checked in", reg.Status))
	}

	// Fetch event and check phase
	var event models.Event
	err = db.NewSelect().Model(&event).Where("id = ?", reg.EventID).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	graceMinutes := 120
	if event.CheckInGraceMinutes != nil {
		graceMinutes = int(*event.CheckInGraceMinutes)
	}
	phase := models.DeriveCheckInPhase(event.Status, time.Now(), event.StartTime, event.EndTime, graceMinutes)
	if phase == models.CheckInPhaseNone {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("check-in is not active for this event yet"))
	}
	if phase == models.CheckInPhaseLocked {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("check-in is locked for this event"))
	}

	// Check if already has attendance record
	var existingAtt models.EventAttendance
	err = db.NewSelect().Model(&existingAtt).Where("registration_id = ?", req.Msg.RegistrationId).Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	method := models.CheckInMethodManual
	var att models.EventAttendance
	var notes *string
	if req.Msg.Notes != nil {
		notes = req.Msg.Notes
	}

	if err == nil {
		// Existing record - update
		att = existingAtt
		att.Status = models.AttendanceStatusCheckedIn
		att.Notes = notes
		att.CheckedInAt = timePtr(time.Now())
		att.CheckedInBy = &req.Msg.CheckedInBy
		att.Method = &method
		att.UpdatedAt = time.Now()
		_, err = db.NewUpdate().
			Model(&att).
			Column("status", "notes", "checked_in_at", "checked_in_by", "method", "updated_at").
			Where("id = ?", att.ID).
			Returning("*").
			Exec(ctx)
	} else {
		// New record - create
		att = models.EventAttendance{
			RegistrationID: req.Msg.RegistrationId,
			Status:         models.AttendanceStatusCheckedIn,
			CheckedInAt:    timePtr(time.Now()),
			CheckedInBy:    &req.Msg.CheckedInBy,
			Notes:          notes,
			Method:         &method,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err = db.NewInsert().Model(&att).Returning("*").Exec(ctx)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.CheckInAttendeeResponse{
		Attendance: dbEventAttendanceToProto(att),
	}), nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}
