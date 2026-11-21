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
)

func (s *Service) GetCheckInInfo(ctx context.Context, req *connect.Request[eventsv1.GetCheckInInfoRequest]) (*connect.Response[eventsv1.GetCheckInInfoResponse], error) {
	slog.Debug("GetCheckInInfo", "eventId", req.Msg.EventId)

	_, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "check_in")
	if err != nil {
		return nil, err
	}

	var event models.Event
	err = s.db.NewSelect().Model(&event).Where("id = ?", req.Msg.EventId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	graceMinutes := 120
	if event.CheckInGraceMinutes != nil {
		graceMinutes = int(*event.CheckInGraceMinutes)
	}
	phase := models.DeriveCheckInPhase(event.Status, time.Now(), event.StartTime, event.EndTime, graceMinutes)

	resp := &eventsv1.GetCheckInInfoResponse{
		CheckInPhase: checkInPhaseToProto(phase),
	}

	// Only expose token info when check-in is live or in grace period
	if phase == models.CheckInPhaseLive || phase == models.CheckInPhaseGrace {
		if event.CheckInToken != nil {
			resp.CheckInToken = event.CheckInToken
		}

		if event.CheckInSecret != nil {
			window := getCurrentWindow()
			currentToken := computeRotatingToken(*event.CheckInSecret, window)
			tokenValidUntil := getWindowEndTime(window).Format(time.RFC3339)

			resp.CurrentRotatingToken = &currentToken
			resp.TokenValidUntil = &tokenValidUntil
			resp.RotationIntervalSeconds = rotationIntervalSeconds
		}

		if event.CheckInToken != nil {
			checkInURL := fmt.Sprintf("%s/check-in/%s", s.mapAppURL, *event.CheckInToken)
			resp.CheckInUrl = &checkInURL
		}
	}

	return connect.NewResponse(resp), nil
}
