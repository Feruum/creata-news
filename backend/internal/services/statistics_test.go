package services_test

import (
	"connectrpc.com/connect"
	"context"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/services/statistics"
	"testing"
)

func TestGetOrganizationTrendsEmptyScopeStillRequiresAuthentication(t *testing.T) {
	service := statistics.NewService(nil, nil)
	for _, days := range []int32{0, 90} {
		_, err := service.GetOrganizationTrends(context.Background(), connect.NewRequest(&eventsv1.GetOrganizationTrendsRequest{Days: days}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("anonymous empty scope bypassed analytics authorization: %v", err)
		}
	}
}
