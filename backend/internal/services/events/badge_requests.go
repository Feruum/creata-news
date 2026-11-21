package events

import (
	"time"

	"github.com/studyverse/ems-backend/internal/models"
)

const (
	badgeNotRequested = "not_requested"
	badgePending      = "pending"
	badgeApproved     = "approved"
	badgeRejected     = "rejected"
)

func nextBadgeRequestStatus(current string, requested *bool) string {
	if current == "" {
		current = badgeNotRequested
	}
	if requested == nil {
		return current
	}
	if !*requested {
		return badgeNotRequested
	}
	if current == badgeApproved || current == badgePending {
		return current
	}
	return badgePending
}

func applyBadgeRequest(status *string, reviewedBy **int32, reviewedAt **time.Time, requested *bool) {
	next := nextBadgeRequestStatus(*status, requested)
	if next != *status {
		*status = next
		*reviewedBy = nil
		*reviewedAt = nil
	}
}

func applyEventBadgeRequests(event *models.Event, socGpa, iros *bool) {
	applyBadgeRequest(&event.SocGpaBadgeStatus, &event.SocGpaBadgeReviewedBy, &event.SocGpaBadgeReviewedAt, socGpa)
	applyBadgeRequest(&event.IrosBadgeStatus, &event.IrosBadgeReviewedBy, &event.IrosBadgeReviewedAt, iros)
}
