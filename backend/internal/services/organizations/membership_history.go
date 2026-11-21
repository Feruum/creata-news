package organizations

import (
	"context"
	"sort"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

// membershipHistoryRow is a joined struct for enriched history queries.
type membershipHistoryRow struct {
	models.MembershipHistory
	UserDisplayName        *string `bun:"user_display_name"`
	PerformedByDisplayName *string `bun:"performed_by_display_name"`
	OrganizationImageURL   *string `bun:"organization_image_url"`
	RoleKey                *string `bun:"role_key"`
	PreviousRoleKey        *string `bun:"previous_role_key"`
}

func (s *Service) GetUserMembershipHistory(ctx context.Context, req *connect.Request[eventsv1.GetUserMembershipHistoryRequest]) (*connect.Response[eventsv1.GetUserMembershipHistoryResponse], error) {
	// Authorization: self, shared org managers, or staff
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if localUserID != req.Msg.UserId {
		if !s.authz.CheckPlatformStaff(ctx, localUserID) {
			// Check if caller manages any org the target user belongs to
			hasSharedOrg, checkErr := s.checkSharedOrgManager(ctx, localUserID, req.Msg.UserId)
			if checkErr != nil || !hasSharedOrg {
				return nil, connect.NewError(connect.CodePermissionDenied, nil)
			}
		}
	}

	limit := int32(50)
	if req.Msg.Limit != nil && *req.Msg.Limit > 0 && *req.Msg.Limit <= 100 {
		limit = *req.Msg.Limit
	}
	offset := int32(0)
	if req.Msg.Offset != nil && *req.Msg.Offset >= 0 {
		offset = *req.Msg.Offset
	}

	query := s.db.NewSelect().
		TableExpr("membership_history AS mh").
		ColumnExpr("mh.*").
		ColumnExpr("CONCAT(u.first_name, ' ', u.last_name) AS user_display_name").
		ColumnExpr("CONCAT(pb.first_name, ' ', pb.last_name) AS performed_by_display_name").
		ColumnExpr("o.image_url AS organization_image_url").
		Join("LEFT JOIN users AS u ON u.id = mh.user_id").
		Join("LEFT JOIN users AS pb ON pb.id = mh.performed_by").
		Join("LEFT JOIN organizations AS o ON o.id = mh.organization_id").
		Where("mh.user_id = ?", req.Msg.UserId).
		OrderExpr("mh.effective_at DESC, mh.id DESC")

	if req.Msg.OrganizationId != nil {
		query = query.Where("mh.organization_id = ?", *req.Msg.OrganizationId)
	}

	// Count total
	countQuery := s.db.NewSelect().
		TableExpr("membership_history AS mh").
		Where("mh.user_id = ?", req.Msg.UserId)
	if req.Msg.OrganizationId != nil {
		countQuery = countQuery.Where("mh.organization_id = ?", *req.Msg.OrganizationId)
	}
	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var rows []membershipHistoryRow
	err = query.Limit(int(limit)).Offset(int(offset)).Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	entries := make([]*eventsv1.MembershipHistoryEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, membershipHistoryRowToProto(r))
	}

	return connect.NewResponse(&eventsv1.GetUserMembershipHistoryResponse{
		Entries:    entries,
		TotalCount: int32(totalCount),
	}), nil
}

func (s *Service) GetOrganizationMembershipHistory(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationMembershipHistoryRequest]) (*connect.Response[eventsv1.GetOrganizationMembershipHistoryResponse], error) {
	// Authorization: org managers with view_analytics, or staff
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if !s.authz.CheckPlatformStaff(ctx, localUserID) {
		if !s.authz.CheckOrgPermission(ctx, localUserID, req.Msg.OrganizationId, "view_analytics") {
			return nil, connect.NewError(connect.CodePermissionDenied, nil)
		}
	}

	limit := int32(50)
	if req.Msg.Limit != nil && *req.Msg.Limit > 0 && *req.Msg.Limit <= 100 {
		limit = *req.Msg.Limit
	}
	offset := int32(0)
	if req.Msg.Offset != nil && *req.Msg.Offset >= 0 {
		offset = *req.Msg.Offset
	}

	query := s.db.NewSelect().
		TableExpr("membership_history AS mh").
		ColumnExpr("mh.*").
		ColumnExpr("CONCAT(u.first_name, ' ', u.last_name) AS user_display_name").
		ColumnExpr("CONCAT(pb.first_name, ' ', pb.last_name) AS performed_by_display_name").
		ColumnExpr("o.image_url AS organization_image_url").
		Join("LEFT JOIN users AS u ON u.id = mh.user_id").
		Join("LEFT JOIN users AS pb ON pb.id = mh.performed_by").
		Join("LEFT JOIN organizations AS o ON o.id = mh.organization_id").
		Where("mh.organization_id = ?", req.Msg.OrganizationId).
		OrderExpr("mh.effective_at DESC, mh.id DESC")

	if req.Msg.ActionFilter != nil && *req.Msg.ActionFilter != "" {
		query = query.Where("mh.action = ?", *req.Msg.ActionFilter)
	}

	countQuery := s.db.NewSelect().
		TableExpr("membership_history AS mh").
		Where("mh.organization_id = ?", req.Msg.OrganizationId)
	if req.Msg.ActionFilter != nil && *req.Msg.ActionFilter != "" {
		countQuery = countQuery.Where("mh.action = ?", *req.Msg.ActionFilter)
	}
	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var rows []membershipHistoryRow
	err = query.Limit(int(limit)).Offset(int(offset)).Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	entries := make([]*eventsv1.MembershipHistoryEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, membershipHistoryRowToProto(r))
	}

	return connect.NewResponse(&eventsv1.GetOrganizationMembershipHistoryResponse{
		Entries:    entries,
		TotalCount: int32(totalCount),
	}), nil
}

func (s *Service) GetUserMembershipTimeline(ctx context.Context, req *connect.Request[eventsv1.GetUserMembershipTimelineRequest]) (*connect.Response[eventsv1.GetUserMembershipTimelineResponse], error) {
	// Public endpoint — no authorization required

	var rows []membershipHistoryRow
	err := s.db.NewSelect().
		TableExpr("membership_history AS mh").
		ColumnExpr("mh.*").
		ColumnExpr("o.image_url AS organization_image_url").
		ColumnExpr("rt.key AS role_key").
		ColumnExpr("prt.key AS previous_role_key").
		Join("LEFT JOIN organizations AS o ON o.id = mh.organization_id").
		Join("LEFT JOIN role_templates AS rt ON rt.id = mh.role_template_id").
		Join("LEFT JOIN role_templates AS prt ON prt.id = mh.previous_role_template_id").
		Where("mh.user_id = ?", req.Msg.UserId).
		OrderExpr("mh.organization_id, mh.effective_at ASC, mh.id ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	tenures := computeTenures(rows)

	if !req.Msg.IncludeEnded {
		filtered := make([]*eventsv1.MembershipTenure, 0)
		for _, t := range tenures {
			if t.IsCurrent {
				filtered = append(filtered, t)
			}
		}
		tenures = filtered
	}

	return connect.NewResponse(&eventsv1.GetUserMembershipTimelineResponse{
		Tenures: tenures,
	}), nil
}

func (s *Service) GetRoleLineage(ctx context.Context, req *connect.Request[eventsv1.GetRoleLineageRequest]) (*connect.Response[eventsv1.GetRoleLineageResponse], error) {
	// Public endpoint — no authorization required

	// Get role template ID from key + org
	var roleTemplate models.RoleTemplate
	err := s.db.NewSelect().Model(&roleTemplate).
		Where("organization_id = ? AND key = ?", req.Msg.OrganizationId, req.Msg.RoleKey).
		Scan(ctx)
	if err != nil {
		// Try org type templates
		var org models.Organization
		if orgErr := s.db.NewSelect().Model(&org).Column("organization_type_id").Where("id = ?", req.Msg.OrganizationId).Scan(ctx); orgErr != nil {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		err = s.db.NewSelect().Model(&roleTemplate).
			Where("organization_type_id = ? AND organization_id IS NULL AND key = ?", org.OrganizationTypeID, req.Msg.RoleKey).
			Scan(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
	}

	var rows []membershipHistoryRow
	err = s.db.NewSelect().
		TableExpr("membership_history AS mh").
		ColumnExpr("mh.*").
		ColumnExpr("o.image_url AS organization_image_url").
		ColumnExpr("rt.key AS role_key").
		ColumnExpr("prt.key AS previous_role_key").
		Join("LEFT JOIN organizations AS o ON o.id = mh.organization_id").
		Join("LEFT JOIN role_templates AS rt ON rt.id = mh.role_template_id").
		Join("LEFT JOIN role_templates AS prt ON prt.id = mh.previous_role_template_id").
		Where("mh.organization_id = ?", req.Msg.OrganizationId).
		Where("(mh.role_template_id = ? OR mh.previous_role_template_id = ?)", roleTemplate.ID, roleTemplate.ID).
		OrderExpr("mh.user_id, mh.effective_at ASC, mh.id ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	tenures := computeTenures(rows)

	// Filter to only tenures for the requested role
	filtered := make([]*eventsv1.MembershipTenure, 0)
	for _, t := range tenures {
		if t.RoleKey == req.Msg.RoleKey {
			filtered = append(filtered, t)
		}
	}

	// Sort by started_at descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartedAt > filtered[j].StartedAt
	})

	return connect.NewResponse(&eventsv1.GetRoleLineageResponse{
		Holders: filtered,
	}), nil
}

// --- Helper functions ---

// checkSharedOrgManager checks if callerID manages any org that targetUserID belongs to.
func (s *Service) checkSharedOrgManager(ctx context.Context, callerID, targetUserID int32) (bool, error) {
	// Get all org IDs the target user belongs to
	var targetOrgIDs []int32
	err := s.db.NewSelect().
		TableExpr("organization_members AS om").
		Column("om.organization_id").
		Where("om.user_id = ?", targetUserID).
		Scan(ctx, &targetOrgIDs)
	if err != nil {
		return false, err
	}
	if len(targetOrgIDs) == 0 {
		return false, nil
	}

	// Check if caller has manage_members permission in any of those orgs
	for _, orgID := range targetOrgIDs {
		if s.authz.CheckOrgPermission(ctx, callerID, orgID, "manage_members") {
			return true, nil
		}
	}
	return false, nil
}

// computeTenures walks history rows and produces tenure periods.
// Rows MUST be ordered by (organization_id, effective_at ASC, id ASC).
func computeTenures(rows []membershipHistoryRow) []*eventsv1.MembershipTenure {
	type tenureKey struct {
		userID int32
		orgID  int32
		roleID int32
	}
	type openTenure struct {
		userID    int32
		orgID     int32
		orgTitle  string
		orgImage  *string
		roleName  string
		roleKey   string
		startedAt time.Time
	}

	open := make(map[tenureKey]*openTenure)
	var result []*eventsv1.MembershipTenure

	closeTenure := func(key tenureKey, endedAt time.Time, endReason string) {
		if t, ok := open[key]; ok {
			ended := endedAt.Format(time.RFC3339)
			tenure := &eventsv1.MembershipTenure{
				UserId:            t.userID,
				OrganizationTitle: t.orgTitle,
				RoleName:          t.roleName,
				RoleKey:           t.roleKey,
				StartedAt:         t.startedAt.Format(time.RFC3339),
				EndedAt:           &ended,
				EndReason:         &endReason,
				IsCurrent:         false,
			}
			if t.orgID != 0 {
				tenure.OrganizationId = &t.orgID
			}
			tenure.OrganizationImageUrl = t.orgImage
			result = append(result, tenure)
			delete(open, key)
		}
	}

	for _, row := range rows {
		orgID := int32(0)
		if row.OrganizationID != nil {
			orgID = *row.OrganizationID
		}

		switch models.MembershipHistoryAction(row.Action) {
		case models.MembershipHistoryActionAssigned:
			if row.RoleTemplateID != nil {
				roleKey := ""
				if row.RoleKey != nil {
					roleKey = *row.RoleKey
				}
				roleName := ""
				if row.RoleName != nil {
					roleName = *row.RoleName
				}
				key := tenureKey{userID: row.UserID, orgID: orgID, roleID: *row.RoleTemplateID}
				open[key] = &openTenure{
					userID:    row.UserID,
					orgID:     orgID,
					orgTitle:  row.OrganizationTitle,
					orgImage:  row.OrganizationImageURL,
					roleName:  roleName,
					roleKey:   roleKey,
					startedAt: row.EffectiveAt,
				}
			}

		case models.MembershipHistoryActionRoleChanged:
			// Close old role tenure
			if row.PreviousRoleTemplateID != nil {
				closeTenure(tenureKey{userID: row.UserID, orgID: orgID, roleID: *row.PreviousRoleTemplateID}, row.EffectiveAt, "role_changed")
			}
			// Open new role tenure
			if row.RoleTemplateID != nil {
				roleKey := ""
				if row.RoleKey != nil {
					roleKey = *row.RoleKey
				}
				roleName := ""
				if row.RoleName != nil {
					roleName = *row.RoleName
				}
				key := tenureKey{userID: row.UserID, orgID: orgID, roleID: *row.RoleTemplateID}
				open[key] = &openTenure{
					userID:    row.UserID,
					orgID:     orgID,
					orgTitle:  row.OrganizationTitle,
					orgImage:  row.OrganizationImageURL,
					roleName:  roleName,
					roleKey:   roleKey,
					startedAt: row.EffectiveAt,
				}
			}

		case models.MembershipHistoryActionRemoved, models.MembershipHistoryActionOrgDeleted, models.MembershipHistoryActionOrgArchived:
			// Close all open tenures for this user+org
			reason := string(row.Action)
			for key := range open {
				if key.userID == row.UserID && key.orgID == orgID {
					closeTenure(key, row.EffectiveAt, reason)
				}
			}
		}
	}

	// Close remaining open tenures as current
	for _, t := range open {
		tenure := &eventsv1.MembershipTenure{
			UserId:            t.userID,
			OrganizationTitle: t.orgTitle,
			RoleName:          t.roleName,
			RoleKey:           t.roleKey,
			StartedAt:         t.startedAt.Format(time.RFC3339),
			IsCurrent:         true,
		}
		if t.orgID != 0 {
			tenure.OrganizationId = &t.orgID
		}
		tenure.OrganizationImageUrl = t.orgImage
		result = append(result, tenure)
	}

	return result
}

func membershipHistoryRowToProto(r membershipHistoryRow) *eventsv1.MembershipHistoryEntry {
	entry := &eventsv1.MembershipHistoryEntry{
		Id:                     r.ID,
		UserId:                 r.UserID,
		OrganizationId:         r.OrganizationID,
		Action:                 string(r.Action),
		RoleTemplateId:         r.RoleTemplateID,
		PreviousRoleTemplateId: r.PreviousRoleTemplateID,
		PerformedBy:            r.PerformedBy,
		Reason:                 r.Reason,
		RoleName:               r.RoleName,
		PreviousRoleName:       r.PreviousRoleName,
		OrganizationTitle:      r.OrganizationTitle,
		EffectiveAt:            r.EffectiveAt.Format(time.RFC3339),
		CreatedAt:              r.CreatedAt.Format(time.RFC3339),
	}
	entry.UserDisplayName = r.UserDisplayName
	entry.PerformedByDisplayName = r.PerformedByDisplayName
	entry.OrganizationImageUrl = r.OrganizationImageURL
	return entry
}
