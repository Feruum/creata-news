//go:build integration

package orgmembers

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func TestProfileOrganizationsReturnCurrentSlugAndActiveMemberships(t *testing.T) {
	db := newTestDB(t)
	userID := seedAuthUser(t, db, "profile-clubs")
	orgID, roleID, _, _ := seedOrgAndRole(t, db)
	memberID := seedOrganizationMember(t, db, orgID, userID, roleID, userID)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "INSERT INTO organization_slugs(organization_id,slug,is_current) VALUES(?,'old-club',false),(?,'current-club',true)", orgID, orgID); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, nil, nil)
	request := connect.NewRequest(&eventsv1.GetUserOrganizationsWithRoleRequest{UserId: userID})
	response, err := svc.GetUserOrganizationsWithRole(ctx, request)
	if err != nil || len(response.Msg.Organizations) != 1 || response.Msg.Organizations[0].Organization.Slug != "current-club" {
		t.Fatalf("profile club link has no current slug: %v, %v", response, err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM organization_slugs WHERE organization_id=?", orgID); err != nil {
		t.Fatal(err)
	}
	response, err = svc.GetUserOrganizationsWithRole(ctx, request)
	if err != nil || len(response.Msg.Organizations) != 1 || response.Msg.Organizations[0].Organization.Id != orgID || response.Msg.Organizations[0].Organization.Slug != "" {
		t.Fatalf("organization without slug is unavailable for ID fallback: %v, %v", response, err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE organization_members SET valid_until=NOW()-interval '1 day' WHERE id=?", memberID); err != nil {
		t.Fatal(err)
	}
	response, err = svc.GetUserOrganizationsWithRole(ctx, request)
	if err != nil || len(response.Msg.Organizations) != 0 {
		t.Fatalf("expired membership is visible: %v, %v", response, err)
	}
}
