package eventattendance

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/checkin"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) SelfCheckIn(ctx context.Context, req *connect.Request[eventsv1.SelfCheckInRequest]) (*connect.Response[eventsv1.SelfCheckInResponse], error) {
	if auth.IsSessionValidationUnavailable(ctx) {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("university sign-in is temporarily unavailable"))
	}
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}

	var localUser models.User
	err = s.db.NewSelect().Model(&localUser).Where("id = ?", localUserID).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Lookup event by check_in_token
	var event models.Event
	err = s.db.NewSelect().Model(&event).Where("check_in_token = ?", req.Msg.Token).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return connect.NewResponse(&eventsv1.SelfCheckInResponse{
				Success: false,
				Message: "Invalid or expired check-in code",
			}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Derive check-in phase from event timestamps
	graceMinutes := 120
	if event.CheckInGraceMinutes != nil {
		graceMinutes = int(*event.CheckInGraceMinutes)
	}
	phase := models.DeriveCheckInPhase(event.Status, time.Now(), event.StartTime, event.EndTime, graceMinutes)

	if phase == models.CheckInPhaseNone || phase == models.CheckInPhaseLocked || !checkin.Active(&event, time.Now()) {
		return connect.NewResponse(&eventsv1.SelfCheckInResponse{
			Success:    false,
			Message:    "Check-in is not available for this event right now",
			EventTitle: &event.Title,
		}), nil
	}

	// Validate rotating token against event secret
	if event.CheckInSecret == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("event has no check-in secret"))
	}
	if !validateRotatingToken(*event.CheckInSecret, req.Msg.RotatingToken) {
		return connect.NewResponse(&eventsv1.SelfCheckInResponse{
			Success:    false,
			Message:    "QR code has expired. Please scan the updated code.",
			EventTitle: &event.Title,
		}), nil
	}
	return s.selfCheckInEvent(ctx, &event, localUser.ID, req.Msg.RotatingToken)
}
