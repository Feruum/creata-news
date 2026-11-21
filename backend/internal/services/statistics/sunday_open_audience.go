package statistics

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetSundayOpenAudienceStatistics(ctx context.Context, req *connect.Request[eventsv1.GetSundayOpenAudienceStatisticsRequest]) (*connect.Response[eventsv1.GetSundayOpenAudienceStatisticsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	var rows []struct {
		Kind  string
		Key   string
		Count int32
	}
	// All counts use one snapshot. Unnest each answer group independently so
	// selecting several interests, goals or formats cannot multiply other counts.
	err := s.db.NewRaw(`WITH completed AS (
		SELECT o.account_id,o.role,o.interests,o.goals,o.formats,o.source
		FROM sunday_open_onboarding o JOIN sunday_open_accounts a ON a.id=o.account_id
		WHERE o.completed_at IS NOT NULL
	)
	SELECT 'total' AS kind,'accounts' AS key,count(*)::integer AS count FROM sunday_open_accounts
	UNION ALL SELECT 'total','completed',count(*)::integer FROM completed
	UNION ALL SELECT 'roles',role,count(*)::integer FROM completed GROUP BY role
	UNION ALL SELECT 'interests',key,count(*)::integer FROM (SELECT DISTINCT account_id,unnest(interests) AS key FROM completed) answers GROUP BY key
	UNION ALL SELECT 'goals',key,count(*)::integer FROM (SELECT DISTINCT account_id,unnest(goals) AS key FROM completed) answers GROUP BY key
	UNION ALL SELECT 'formats',key,count(*)::integer FROM (SELECT DISTINCT account_id,unnest(formats) AS key FROM completed) answers GROUP BY key
	UNION ALL SELECT 'sources',source,count(*)::integer FROM completed GROUP BY source
	ORDER BY kind,key`).Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load Sunday Open audience statistics: %w", err))
	}

	response := &eventsv1.GetSundayOpenAudienceStatisticsResponse{}
	for _, row := range rows {
		bucket := &eventsv1.SundayOpenAudienceBucket{Key: row.Key, Count: row.Count}
		switch row.Kind {
		case "total":
			if row.Key == "accounts" {
				response.TotalAccounts = row.Count
			} else {
				response.CompletedAccounts = row.Count
			}
		case "roles":
			response.Roles = append(response.Roles, bucket)
		case "interests":
			response.Interests = append(response.Interests, bucket)
		case "goals":
			response.Goals = append(response.Goals, bucket)
		case "formats":
			response.Formats = append(response.Formats, bucket)
		case "sources":
			response.Sources = append(response.Sources, bucket)
		}
	}
	return connect.NewResponse(response), nil
}
