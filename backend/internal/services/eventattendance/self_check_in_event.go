package eventattendance

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/checkin"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

// A valid live QR confirms admission, including for walk-ins. Registration and
// attendance commit together, serialized with registration review/cancellation.
func (s *Service) selfCheckInEvent(ctx context.Context, event *models.Event, userID int32, code string) (*connect.Response[eventsv1.SelfCheckInResponse], error) {
	already := false
	scannedToken := ""
	if event.CheckInToken != nil {
		scannedToken = *event.CheckInToken
	}
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewSelect().Model(event).WherePK().For("NO KEY UPDATE").Scan(ctx); err != nil {
			return err
		}
		if scannedToken == "" || event.CheckInToken == nil || *event.CheckInToken != scannedToken {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("QR code was revoked. Scan the updated code."))
		}
		if !checkin.Active(event, time.Now()) {
			return connect.NewError(connect.CodeFailedPrecondition, errors.New("check-in is closed"))
		}
		if event.CheckInSecret == nil || !checkin.ValidCode(*event.CheckInSecret, code, time.Now()) {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("QR code has expired. Scan the updated code."))
		}
		var reg models.EventRegistration
		err := tx.NewSelect().Model(&reg).Where("event_id = ?", event.ID).Where("user_id = ?", userID).Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if errors.Is(err, sql.ErrNoRows) || reg.Status != models.RegistrationStatusRegistered {
			if reg.ID == 0 {
				if err := tx.NewRaw(`INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,'registered') RETURNING id`, event.ID, userID).Scan(ctx, &reg.ID); err != nil {
					return err
				}
			} else if _, err := tx.ExecContext(ctx, `UPDATE event_registrations SET status='registered',cancelled_at=NULL,registered_at=NOW(),updated_at=NOW() WHERE id=?`, reg.ID); err != nil {
				return err
			}
		}
		var attended bool
		if err := tx.NewRaw(`SELECT EXISTS(SELECT 1 FROM event_attendance WHERE registration_id=? AND status IN ('checked_in','attended'))`, reg.ID).Scan(ctx, &attended); err != nil {
			return err
		}
		if attended {
			already = true
			return nil
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO event_attendance(registration_id,status,checked_in_at,checked_in_by,method) VALUES(?,'checked_in',NOW(),?,'self') ON CONFLICT(registration_id) DO UPDATE SET status='checked_in',checked_in_at=COALESCE(event_attendance.checked_in_at,EXCLUDED.checked_in_at),checked_in_by=EXCLUDED.checked_in_by,method='self',updated_at=NOW() WHERE event_attendance.status NOT IN ('checked_in','attended')`, reg.ID, userID)
		return err
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeUnknown {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return nil, err
	}
	message := "Checked in successfully!"
	if already {
		message = "You're already checked in!"
	}
	return connect.NewResponse(&eventsv1.SelfCheckInResponse{Success: true, Message: message, EventTitle: &event.Title}), nil
}
