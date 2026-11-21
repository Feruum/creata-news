package events

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestBadgeRequestsRequireAdministratorReview(t *testing.T) {
	requested, removed := true, false
	for _, tc := range []struct {
		name      string
		current   string
		requested *bool
		want      string
	}{
		{"old client preserves approval", "approved", nil, "approved"},
		{"old client preserves rejection", "rejected", nil, "rejected"},
		{"first request needs review", "not_requested", &requested, "pending"},
		{"unchanged request remains pending", "pending", &requested, "pending"},
		{"unchanged approved selection remains approved", "approved", &requested, "approved"},
		{"request after rejection needs review", "rejected", &requested, "pending"},
		{"remove approved request", "approved", &removed, "not_requested"},
		{"remove pending request", "pending", &removed, "not_requested"},
		{"removed request cannot inherit approval", "not_requested", &requested, "pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextBadgeRequestStatus(tc.current, tc.requested); got != tc.want {
				t.Fatalf("badge state = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBadgeRemovalClearsReviewAndPreservesOtherBadge(t *testing.T) {
	reviewer := int32(7)
	reviewedAt := time.Now()
	event := models.Event{
		SocGpaBadgeStatus: badgeApproved, SocGpaBadgeReviewedBy: &reviewer, SocGpaBadgeReviewedAt: &reviewedAt,
		IrosBadgeStatus: badgeApproved, IrosBadgeReviewedBy: &reviewer, IrosBadgeReviewedAt: &reviewedAt,
	}
	removed, requested := false, true
	applyEventBadgeRequests(&event, &removed, nil)
	if event.SocGpaBadgeStatus != badgeNotRequested || event.SocGpaBadgeReviewedBy != nil || event.SocGpaBadgeReviewedAt != nil {
		t.Fatal("removed badge retained its approval")
	}
	if event.IrosBadgeStatus != badgeApproved || event.IrosBadgeReviewedBy != &reviewer || event.IrosBadgeReviewedAt != &reviewedAt {
		t.Fatal("removal changed the other badge")
	}
	applyEventBadgeRequests(&event, &requested, nil)
	if event.SocGpaBadgeStatus != badgePending || event.SocGpaBadgeReviewedBy != nil {
		t.Fatal("re-request inherited the previous approval")
	}
}

func TestBadgeReviewOnlyAcceptsDecisionsForRequestedBadges(t *testing.T) {
	approved := eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_APPROVED
	rejected := eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_REJECTED
	pending := eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_PENDING
	for _, tc := range []struct {
		name     string
		current  string
		decision *eventsv1.EventBadgeStatus
		code     connect.Code
	}{
		{"approve pending", badgePending, &approved, 0},
		{"reject pending", badgePending, &rejected, 0},
		{"correct rejection", badgeRejected, &approved, 0},
		{"remove approval", badgeApproved, &rejected, 0},
		{"omitted decision", badgeNotRequested, nil, 0},
		{"unrequested badge", badgeNotRequested, &approved, connect.CodeFailedPrecondition},
		{"invalid decision", badgePending, &pending, connect.CodeInvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBadgeReview(tc.current, tc.decision)
			if tc.code == 0 && err != nil || tc.code != 0 && connect.CodeOf(err) != tc.code {
				t.Fatalf("review error = %v, want code %v", err, tc.code)
			}
		})
	}
}

func TestBadgeReviewRequiresAuthentication(t *testing.T) {
	svc := &Service{}
	_, err := svc.ReviewEventBadges(context.Background(), connect.NewRequest(&eventsv1.ReviewEventBadgesRequest{EventId: 1}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated review returned %v", err)
	}
}
