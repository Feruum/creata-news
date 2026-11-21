//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestHandleMemberAddedHistoryInsertsMembershipHistoryRow(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	handler := handleMemberAddedHistory(HandlerDeps{DB: db})

	performedBy := int32(99)
	payload, _ := json.Marshal(events.MemberAddedPayload{
		UserID:            1,
		OrganizationID:    10,
		OrganizationTitle: "Chess Club",
		RoleTemplateID:    5,
		RoleName:          "member",
		PerformedBy:       &performedBy,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(ctx)

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	var row models.MembershipHistory
	err := db.NewSelect().Model(&row).
		Where("user_id = ? AND organization_id = ?", 1, 10).
		Scan(ctx)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if row.Action != models.MembershipHistoryActionAssigned {
		t.Fatalf("action = %q, want %q", row.Action, models.MembershipHistoryActionAssigned)
	}
	if row.RoleTemplateID == nil || *row.RoleTemplateID != 5 {
		t.Fatalf("role_template_id = %v, want 5", row.RoleTemplateID)
	}
	if row.RoleName == nil || *row.RoleName != "member" {
		t.Fatalf("role_name = %v, want %q", row.RoleName, "member")
	}
	if row.PerformedBy == nil || *row.PerformedBy != 99 {
		t.Fatalf("performed_by = %v, want 99", row.PerformedBy)
	}
	if row.OrganizationTitle != "Chess Club" {
		t.Fatalf("organization_title = %q, want %q", row.OrganizationTitle, "Chess Club")
	}
}

func TestHandleMemberRemovedHistorySetsPreviousRole(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	handler := handleMemberRemovedHistory(HandlerDeps{DB: db})

	performedBy := int32(42)
	payload, _ := json.Marshal(events.MemberRemovedPayload{
		UserID:            2,
		OrganizationID:    11,
		OrganizationTitle: "Drama Club",
		RoleTemplateID:    7,
		RoleName:          "president",
		PerformedBy:       &performedBy,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(ctx)

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	var row models.MembershipHistory
	err := db.NewSelect().Model(&row).
		Where("user_id = ? AND organization_id = ?", 2, 11).
		Scan(ctx)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if row.Action != models.MembershipHistoryActionRemoved {
		t.Fatalf("action = %q, want %q", row.Action, models.MembershipHistoryActionRemoved)
	}
	if row.PreviousRoleTemplateID == nil || *row.PreviousRoleTemplateID != 7 {
		t.Fatalf("previous_role_template_id = %v, want 7", row.PreviousRoleTemplateID)
	}
	if row.PreviousRoleName == nil || *row.PreviousRoleName != "president" {
		t.Fatalf("previous_role_name = %v, want %q", row.PreviousRoleName, "president")
	}
	if row.PerformedBy == nil || *row.PerformedBy != 42 {
		t.Fatalf("performed_by = %v, want 42", row.PerformedBy)
	}
}
