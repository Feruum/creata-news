package eventattendance

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) StaffOverrideCheckIn(ctx context.Context, req *connect.Request[eventsv1.StaffOverrideCheckInRequest]) (*connect.Response[eventsv1.StaffOverrideCheckInResponse], error) {
	return attendanceMutation(ctx, s, req.Msg.RegistrationId, true, func(ctx context.Context, db bun.IDB, actor int32) (*connect.Response[eventsv1.StaffOverrideCheckInResponse], error) {
		return s.staffOverrideCheckIn(ctx, db, req, actor)
	})
}
func (s *Service) staffOverrideCheckIn(ctx context.Context, db bun.IDB, req *connect.Request[eventsv1.StaffOverrideCheckInRequest], localUserID int32) (*connect.Response[eventsv1.StaffOverrideCheckInResponse], error) {
	slog.Debug("StaffOverrideCheckIn", "registrationId", req.Msg.RegistrationId)

	var reg models.EventRegistration
	err := db.NewSelect().Model(&reg).Where("id = ?", req.Msg.RegistrationId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("registration not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if reg.Status != models.RegistrationStatusRegistered {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("only confirmed registrations may be checked in"))
	}
	method := models.CheckInMethodStaffOverride
	var notes *string
	if req.Msg.Notes != nil {
		notes = req.Msg.Notes
	}

	var existingAtt models.EventAttendance
	err = db.NewSelect().Model(&existingAtt).Where("registration_id = ?", req.Msg.RegistrationId).Scan(ctx)

	var att models.EventAttendance
	if err == nil {
		att = existingAtt
		att.Status = models.AttendanceStatusCheckedIn
		att.CheckedInAt = timePtr(time.Now())
		att.CheckedInBy = &localUserID
		att.Method = &method
		att.Notes = notes
		att.UpdatedAt = time.Now()
		_, err = db.NewUpdate().
			Model(&att).
			Column("status", "checked_in_at", "checked_in_by", "method", "notes", "updated_at").
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

	return connect.NewResponse(&eventsv1.StaffOverrideCheckInResponse{
		Attendance: dbEventAttendanceToProto(att),
	}), nil
}
