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

func (s *Service) MarkAttendance(ctx context.Context, req *connect.Request[eventsv1.MarkAttendanceRequest]) (*connect.Response[eventsv1.MarkAttendanceResponse], error) {
	return attendanceMutation(ctx, s, req.Msg.RegistrationId, false, func(ctx context.Context, db bun.IDB, actor int32) (*connect.Response[eventsv1.MarkAttendanceResponse], error) {
		return s.markAttendance(ctx, db, req)
	})
}
func (s *Service) markAttendance(ctx context.Context, db bun.IDB, req *connect.Request[eventsv1.MarkAttendanceRequest]) (*connect.Response[eventsv1.MarkAttendanceResponse], error) {
	slog.Debug("MarkAttendance", "registrationId", req.Msg.RegistrationId, "status", req.Msg.Status)

	var reg models.EventRegistration
	err := db.NewSelect().Model(&reg).Where("id = ?", req.Msg.RegistrationId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if reg.Status != models.RegistrationStatusRegistered {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("only confirmed registrations may have attendance marked"))
	}

	status := protoAttendanceStatusToDB(req.Msg.Status)

	var existingAtt models.EventAttendance
	err = db.NewSelect().Model(&existingAtt).Where("registration_id = ?", req.Msg.RegistrationId).Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var att models.EventAttendance
	var notes *string
	if req.Msg.Notes != nil {
		notes = req.Msg.Notes
	}

	if err == nil {
		att = existingAtt
		att.Status = status
		att.Notes = notes
		att.UpdatedAt = time.Now()
		if status == models.AttendanceStatusCheckedIn || status == models.AttendanceStatusAttended {
			att.CheckedInAt = timePtr(time.Now())
		}
		_, err = db.NewUpdate().
			Model(&att).
			Column("status", "notes", "checked_in_at", "updated_at").
			Where("id = ?", att.ID).
			Returning("*").
			Exec(ctx)
	} else {
		markCheckedIn := status == models.AttendanceStatusCheckedIn || status == models.AttendanceStatusAttended
		att = models.EventAttendance{
			RegistrationID: req.Msg.RegistrationId,
			Status:         status,
			Notes:          notes,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if markCheckedIn {
			att.CheckedInAt = timePtr(time.Now())
		}
		_, err = db.NewInsert().Model(&att).Returning("*").Exec(ctx)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.MarkAttendanceResponse{
		Attendance: dbEventAttendanceToProto(att),
	}), nil
}
