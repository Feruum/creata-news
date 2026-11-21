package events

import (
	"context"
	"github.com/uptrace/bun"
)

func (s *Service) editableOrganizationIDs(ctx context.Context, userID int32) ([]int32, error) {
	var candidates []int32
	if err := s.db.NewSelect().Table("organization_members").Column("organization_id").Where("user_id=?", userID).Scan(ctx, &candidates); err != nil {
		return nil, err
	}
	allowed := make([]int32, 0, len(candidates))
	for _, orgID := range candidates {
		if s.authz.CheckOrgPermission(ctx, userID, orgID, "create_event") {
			allowed = append(allowed, orgID)
		}
	}
	return allowed, nil
}
func restrictEditableEvents(query *bun.SelectQuery, userID int32, orgIDs []int32) *bun.SelectQuery {
	return query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		q = q.Where("e.user_id = ?", userID)
		if len(orgIDs) > 0 {
			q = q.WhereOr("e.organization_id IN (?)", bun.In(orgIDs))
			q = q.WhereOr("EXISTS(SELECT 1 FROM event_collaborators AS editable_collaborator WHERE editable_collaborator.event_id=e.id AND editable_collaborator.organization_id IN (?))", bun.In(orgIDs))
		}
		return q
	})
}
