package statistics

import (
	"connectrpc.com/connect"
	"context"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/auth"
	"net/http"
	"net/http/httptest"
	"testing"
)

func statisticsCalls(service eventsv1connect.StatisticsServiceHandler) map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"Sunday Open analytics": func(ctx context.Context) error {
			_, err := service.GetSundayOpenAnalytics(ctx, connect.NewRequest(&eventsv1.GetSundayOpenAnalyticsRequest{}))
			return err
		},
		"Sunday Open audience": func(ctx context.Context) error {
			_, err := service.GetSundayOpenAudienceStatistics(ctx, connect.NewRequest(&eventsv1.GetSundayOpenAudienceStatisticsRequest{}))
			return err
		},
		"dashboard": func(ctx context.Context) error {
			_, err := service.GetDashboardStatistics(ctx, connect.NewRequest(&eventsv1.GetDashboardStatisticsRequest{}))
			return err
		},
		"event": func(ctx context.Context) error {
			_, err := service.GetEventStatistics(ctx, connect.NewRequest(&eventsv1.GetEventStatisticsRequest{EventId: 1}))
			return err
		},
		"categories": func(ctx context.Context) error {
			_, err := service.GetEventCategoriesDistributionByMonth(ctx, connect.NewRequest(&eventsv1.GetEventCategoriesDistributionByMonthRequest{Year: 2026, Month: 1}))
			return err
		},
		"year": func(ctx context.Context) error {
			_, err := service.GetEventActivityByYear(ctx, connect.NewRequest(&eventsv1.GetEventActivityByYearRequest{Year: 2026}))
			return err
		},
		"overall": func(ctx context.Context) error {
			_, err := service.GetOverallStatistics(ctx, connect.NewRequest(&eventsv1.GetOverallStatisticsRequest{}))
			return err
		},
		"trends": func(ctx context.Context) error {
			_, err := service.GetEventTrends(ctx, connect.NewRequest(&eventsv1.GetEventTrendsRequest{}))
			return err
		},
		"clubs": func(ctx context.Context) error {
			_, err := service.GetTopPerformingClubs(ctx, connect.NewRequest(&eventsv1.GetTopPerformingClubsRequest{}))
			return err
		},
		"engagement": func(ctx context.Context) error {
			_, err := service.GetUserEngagementLevels(ctx, connect.NewRequest(&eventsv1.GetUserEngagementLevelsRequest{}))
			return err
		},
		"top events": func(ctx context.Context) error {
			_, err := service.GetTopPerformingEvents(ctx, connect.NewRequest(&eventsv1.GetTopPerformingEventsRequest{}))
			return err
		},
		"low registration": func(ctx context.Context) error {
			_, err := service.GetLowRegistrationEvents(ctx, connect.NewRequest(&eventsv1.GetLowRegistrationEventsRequest{}))
			return err
		},
		"org activity": func(ctx context.Context) error {
			_, err := service.GetOrganizationActivity(ctx, connect.NewRequest(&eventsv1.GetOrganizationActivityRequest{}))
			return err
		},
		"org trends": func(ctx context.Context) error {
			_, err := service.GetOrganizationTrends(ctx, connect.NewRequest(&eventsv1.GetOrganizationTrendsRequest{OrganizationIds: []int32{1}}))
			return err
		},
	}
}
func TestEveryStatisticsRPCRejectsAnonymousAndMissingAuthorizerBeforeDatabase(t *testing.T) {
	for _, test := range []struct {
		name string
		ctx  context.Context
		code connect.Code
	}{
		{"anonymous", context.Background(), connect.CodeUnauthenticated},
		{"authenticated missing authorizer", context.WithValue(context.Background(), auth.UserIDKey, "existing-user"), connect.CodeUnavailable},
	} {
		for name, call := range statisticsCalls(&Service{}) {
			t.Run(test.name+"/"+name, func(t *testing.T) {
				defer func() {
					if recovered := recover(); recovered != nil {
						t.Fatalf("statistics reached nil database before rejecting request: %v", recovered)
					}
				}()
				if err := call(test.ctx); connect.CodeOf(err) != test.code {
					t.Fatalf("expected %v, got %v", test.code, err)
				}
			})
		}
	}
}

func TestEveryStatisticsHTTPRPCRejectsAnonymous(t *testing.T) {
	path, handler := eventsv1connect.NewStatisticsServiceHandler(&Service{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := eventsv1connect.NewStatisticsServiceClient(server.Client(), server.URL)
	for name, call := range statisticsCalls(client) {
		t.Run(name, func(t *testing.T) {
			if err := call(context.Background()); connect.CodeOf(err) != connect.CodeUnauthenticated {
				t.Fatalf("anonymous HTTP statistics disclosed data: %v", err)
			}
		})
	}
}
