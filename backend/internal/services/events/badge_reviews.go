package events

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func validateBadgeReview(current string, decision *eventsv1.EventBadgeStatus) error {
	if decision == nil {
		return nil
	}
	if *decision != eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_APPROVED && *decision != eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_REJECTED {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("choose approve or reject for a badge review"))
	}
	if current != badgePending && current != badgeApproved && current != badgeRejected {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("request a badge before reviewing it"))
	}
	return nil
}

func applyBadgeReview(status *string, reviewedBy **int32, reviewedAt **time.Time, decision *eventsv1.EventBadgeStatus, reviewer int32, now time.Time) bool {
	if decision == nil {
		return false
	}
	next := badgeRejected
	if *decision == eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_APPROVED {
		next = badgeApproved
	}
	if *status == next {
		return false
	}
	*status = next
	*reviewedBy = &reviewer
	*reviewedAt = &now
	return true
}

func (s *Service) ReviewEventBadges(ctx context.Context, req *connect.Request[eventsv1.ReviewEventBadgesRequest]) (*connect.Response[eventsv1.ReviewEventBadgesResponse], error) {
	return runEventMutation(ctx, s, func(scoped *Service) (*connect.Response[eventsv1.ReviewEventBadgesResponse], error) {
		reviewer, err := scoped.authz.RequirePlatformAdmin(ctx)
		if err != nil {
			return nil, err
		}
		if req.Msg.SocGpaStatus == nil && req.Msg.IrosStatus == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("choose at least one badge to review"))
		}
		event := &models.Event{ID: req.Msg.EventId}
		if err := scoped.db.NewSelect().Model(event).WherePK().For("UPDATE").Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, errors.New("event not found"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load event badge requests: %w", err))
		}
		if err := validateBadgeReview(event.SocGpaBadgeStatus, req.Msg.SocGpaStatus); err != nil {
			return nil, err
		}
		if err := validateBadgeReview(event.IrosBadgeStatus, req.Msg.IrosStatus); err != nil {
			return nil, err
		}
		now := time.Now()
		socGpaChanged := applyBadgeReview(&event.SocGpaBadgeStatus, &event.SocGpaBadgeReviewedBy, &event.SocGpaBadgeReviewedAt, req.Msg.SocGpaStatus, reviewer, now)
		irosChanged := applyBadgeReview(&event.IrosBadgeStatus, &event.IrosBadgeReviewedBy, &event.IrosBadgeReviewedAt, req.Msg.IrosStatus, reviewer, now)
		if !socGpaChanged && !irosChanged {
			return connect.NewResponse(&eventsv1.ReviewEventBadgesResponse{Event: dbEventToProto(*event, nil, nil)}), nil
		}
		event.UpdatedAt = now
		if _, err := scoped.db.NewUpdate().Model(event).
			Column("soc_gpa_badge_status", "soc_gpa_badge_reviewed_by", "soc_gpa_badge_reviewed_at", "iros_badge_status", "iros_badge_reviewed_by", "iros_badge_reviewed_at", "updated_at").
			WherePK().Exec(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save event badge review: %w", err))
		}
		return connect.NewResponse(&eventsv1.ReviewEventBadgesResponse{Event: scoped.updatedEventProto(ctx, event, reviewer)}), nil
	})
}
