package events

import (
	"time"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func dbBadgeToProto(status string, reviewedBy *int32, reviewedAt *time.Time) *eventsv1.EventBadge {
	protoStatus := eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_UNSPECIFIED
	switch status {
	case "", badgeNotRequested:
		protoStatus = eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_NOT_REQUESTED
	case badgePending:
		protoStatus = eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_PENDING
	case badgeApproved:
		protoStatus = eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_APPROVED
	case badgeRejected:
		protoStatus = eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_REJECTED
	}
	badge := &eventsv1.EventBadge{Status: protoStatus, ReviewedBy: reviewedBy}
	if reviewedAt != nil {
		value := reviewedAt.Format(time.RFC3339)
		badge.ReviewedAt = &value
	}
	return badge
}
