package authz

import (
	"github.com/studyverse/ems-backend/internal/models"
	"testing"
	"time"
)

func TestMembershipEligibilityRejectsRevokedExpiredAndOutOfScopeRoles(t *testing.T) {
	now := time.Now()
	org := models.Organization{ID: 3, OrganizationTypeID: 2}
	member := models.OrganizationMember{OrganizationID: 3}
	template := models.RoleTemplate{OrganizationTypeID: 2, IsActive: true}
	if !membershipEligible(member, template, org, now) {
		t.Fatal("valid member denied")
	}
	expiry := now
	member.ValidUntil = &expiry
	if membershipEligible(member, template, org, now) {
		t.Fatal("expired member allowed")
	}
	member.ValidUntil = nil
	template.IsActive = false
	if membershipEligible(member, template, org, now) {
		t.Fatal("inactive template allowed")
	}
	template.IsActive = true
	other := int32(9)
	template.OrganizationID = &other
	if membershipEligible(member, template, org, now) {
		t.Fatal("foreign org role allowed")
	}
	template.OrganizationID = nil
	template.OrganizationTypeID = 9
	if membershipEligible(member, template, org, now) {
		t.Fatal("foreign org type role allowed")
	}
}
