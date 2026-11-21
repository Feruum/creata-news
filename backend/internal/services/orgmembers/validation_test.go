package orgmembers

import (
	"github.com/studyverse/ems-backend/internal/models"
	"testing"
	"time"
)

func TestRoleAssignmentScopeAndLimits(t *testing.T) {
	org := models.Organization{ID: 3, OrganizationTypeID: 2}
	role := models.RoleTemplate{OrganizationTypeID: 2, IsActive: true}
	if err := validateRoleAssignment(role, org, 0); err != nil {
		t.Fatal(err)
	}
	other := int32(4)
	role.OrganizationID = &other
	if validateRoleAssignment(role, org, 0) == nil {
		t.Fatal("foreign organization role accepted")
	}
	role.OrganizationID = nil
	role.OrganizationTypeID = 4
	if validateRoleAssignment(role, org, 0) == nil {
		t.Fatal("foreign type role accepted")
	}
	role.OrganizationTypeID = 2
	role.IsActive = false
	if validateRoleAssignment(role, org, 0) == nil {
		t.Fatal("inactive role accepted")
	}
	role.IsActive = true
	limit := int32(1)
	role.MaxHolders = &limit
	if validateRoleAssignment(role, org, 1) == nil {
		t.Fatal("full role accepted")
	}
	if err := validateRoleAssignment(role, org, 0); err != nil {
		t.Fatal(err)
	}
}
func TestMembershipExpiryValidation(t *testing.T) {
	now := time.Now()
	for _, input := range []string{"invalid", now.Add(-time.Hour).Format(time.RFC3339), now.Format(time.RFC3339)} {
		if _, err := parseMembershipExpiry(&input, now); err == nil {
			t.Fatalf("invalid expiry %q accepted", input)
		}
	}
	future := now.Add(time.Hour).Format(time.RFC3339)
	if _, err := parseMembershipExpiry(&future, now); err != nil {
		t.Fatal(err)
	}
	empty := ""
	if expiry, err := parseMembershipExpiry(&empty, now); err != nil || expiry != nil {
		t.Fatal("empty expiry must clear the limit")
	}
}
