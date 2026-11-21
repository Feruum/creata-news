package statistics

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"fmt"
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/uptrace/bun"
)

var _ eventsv1connect.StatisticsServiceHandler = (*Service)(nil)

type Service struct {
	eventsv1connect.UnimplementedStatisticsServiceHandler
	db    *bun.DB
	authz *authz.Enforcer
}

func NewService(db *bun.DB, enforcer *authz.Enforcer) *Service {
	return &Service{db: db, authz: enforcer}
}
func (s *Service) requireAuthorizer(ctx context.Context) error {
	if auth.GetUserID(ctx) == "" {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	if s.authz == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("authorization service unavailable"))
	}
	return nil
}
func (s *Service) requirePlatformStaff(ctx context.Context) error {
	if err := s.requireAuthorizer(ctx); err != nil {
		return err
	}
	_, err := s.authz.RequirePlatformStaff(ctx)
	return err
}
func (s *Service) requireEventAnalytics(ctx context.Context, eventID int32) error {
	if err := s.requireAuthorizer(ctx); err != nil {
		return err
	}
	_, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprint(eventID), "view_analytics")
	return err
}
func (s *Service) requireOrganizationAnalytics(ctx context.Context, orgIDs []int32) error {
	if err := s.requireAuthorizer(ctx); err != nil {
		return err
	}
	// Empty scope remains protected and cannot become an implicit global query.
	if len(orgIDs) == 0 {
		_, err := s.authz.GetLocalUserID(ctx)
		return err
	}
	for _, orgID := range orgIDs {
		if orgID <= 0 {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("organization IDs must be positive"))
		}
		if _, err := s.authz.RequireOrgPermission(ctx, orgID, "view_analytics"); err != nil {
			return err
		}
	}
	return nil
}
