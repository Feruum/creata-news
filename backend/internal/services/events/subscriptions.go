package events

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) SubscribeToOrganization(ctx context.Context, req *connect.Request[eventsv1.SubscribeToOrganizationRequest]) (*connect.Response[eventsv1.SubscribeToOrganizationResponse], error) {
	slog.Debug("SubscribeToOrganization", "organizationId", req.Msg.OrganizationId)

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Verify organization exists
	exists, err := s.db.NewSelect().Model((*models.Organization)(nil)).Where("id = ?", req.Msg.OrganizationId).Exists(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check organization: %w", err))
	}
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
	}

	sub := &models.OrganizationSubscription{
		OrganizationID: req.Msg.OrganizationId,
		UserID:         localUserID,
	}

	_, err = s.db.NewInsert().Model(sub).On("CONFLICT (organization_id, user_id) DO NOTHING").Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to subscribe: %w", err))
	}

	return connect.NewResponse(&eventsv1.SubscribeToOrganizationResponse{}), nil
}

func (s *Service) UnsubscribeFromOrganization(ctx context.Context, req *connect.Request[eventsv1.UnsubscribeFromOrganizationRequest]) (*connect.Response[eventsv1.UnsubscribeFromOrganizationResponse], error) {
	slog.Debug("UnsubscribeFromOrganization", "organizationId", req.Msg.OrganizationId)

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	_, err = s.db.NewDelete().Model((*models.OrganizationSubscription)(nil)).
		Where("organization_id = ? AND user_id = ?", req.Msg.OrganizationId, localUserID).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unsubscribe: %w", err))
	}

	return connect.NewResponse(&eventsv1.UnsubscribeFromOrganizationResponse{}), nil
}

func (s *Service) GetOrganizationSubscriptionStatus(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationSubscriptionStatusRequest]) (*connect.Response[eventsv1.GetOrganizationSubscriptionStatusResponse], error) {
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	isSubscribed, err := s.db.NewSelect().Model((*models.OrganizationSubscription)(nil)).
		Where("organization_id = ? AND user_id = ?", req.Msg.OrganizationId, localUserID).
		Exists(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check subscription status: %w", err))
	}

	return connect.NewResponse(&eventsv1.GetOrganizationSubscriptionStatusResponse{
		IsSubscribed: isSubscribed,
	}), nil
}

func (s *Service) GetOrganizationSubscriberCount(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationSubscriberCountRequest]) (*connect.Response[eventsv1.GetOrganizationSubscriberCountResponse], error) {
	count, err := s.db.NewSelect().Model((*models.OrganizationSubscription)(nil)).
		Where("organization_id = ?", req.Msg.OrganizationId).
		Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count subscribers: %w", err))
	}

	return connect.NewResponse(&eventsv1.GetOrganizationSubscriberCountResponse{
		Count: int32(count),
	}), nil
}
