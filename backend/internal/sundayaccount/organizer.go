package sundayaccount

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/checkin"
	"github.com/studyverse/ems-backend/internal/models"
)

type organizerService interface {
	GetCheckInInfo(context.Context, *connect.Request[eventsv1.GetCheckInInfoRequest]) (*connect.Response[eventsv1.GetCheckInInfoResponse], error)
}

func (h *Handler) WithOrganizer(service organizerService) *Handler {
	h.organizer = service
	return h
}

func (h *Handler) organizerCheckIn(w http.ResponseWriter, r *http.Request) {
	claim, err := checkin.Verify(h.cfg.Secret, r.Header.Get("X-Organizer-Ticket"), "organizer", time.Now())
	if err != nil || !checkin.AllowedOrigin(h.cfg.Origin, claim.Origin) ||
		(r.Header.Get("Origin") != "" && r.Header.Get("Origin") != claim.Origin) {
		fail(w, 403, "Authorize QR display through AITU Map")
		return
	}
	if h.organizer == nil {
		fail(w, 503, "QR display is unavailable")
		return
	}
	ctx := context.WithValue(r.Context(), auth.UserIDKey, claim.Subject)
	response, err := h.organizer.GetCheckInInfo(ctx, connect.NewRequest(&eventsv1.GetCheckInInfoRequest{EventId: claim.EventID}))
	if err != nil {
		switch connect.CodeOf(err) {
		case connect.CodeUnauthenticated, connect.CodePermissionDenied:
			fail(w, 403, "You do not have permission to show this QR")
		case connect.CodeNotFound:
			fail(w, 404, "Event not found")
		default:
			fail(w, 503, "Could not load QR")
		}
		return
	}
	var event models.Event
	if err := h.db.NewSelect().Model(&event).Where("id = ?", claim.EventID).Where("publish_on_sunday_open = true").Scan(ctx); err != nil {
		fail(w, 404, "Event not found")
		return
	}
	info := response.Msg
	qrURL := ""
	if info.GetCheckInUrl() != "" && info.GetCurrentRotatingToken() != "" && checkin.Active(&event, time.Now()) {
		qrURL = info.GetCheckInUrl() + "?code=" + info.GetCurrentRotatingToken()
	}
	phase := "not_open"
	switch info.CheckInPhase {
	case eventsv1.CheckInPhase_CHECK_IN_PHASE_LIVE:
		phase = "live"
	case eventsv1.CheckInPhase_CHECK_IN_PHASE_GRACE:
		phase = "grace"
	case eventsv1.CheckInPhase_CHECK_IN_PHASE_LOCKED:
		phase = "locked"
	}
	if event.CompletedAt != nil {
		phase = "locked"
	}
	writeJSON(w, 200, map[string]any{"eventId": event.ID, "title": event.Title, "phase": phase, "url": qrURL, "tokenValidUntil": info.GetTokenValidUntil(), "accessExpiresAt": time.Unix(claim.ExpiresAt, 0).UTC().Format(time.RFC3339)})
}
