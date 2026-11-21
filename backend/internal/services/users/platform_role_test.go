package users

import (
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"testing"
)

func TestPlatformRoleValidation(t *testing.T) {
	for _, test := range []struct {
		role     usersv1.PlatformRole
		relation string
	}{{usersv1.PlatformRole_PLATFORM_ROLE_ADMIN, "administrator"}, {usersv1.PlatformRole_PLATFORM_ROLE_STAFF, "staff"}, {usersv1.PlatformRole_PLATFORM_ROLE_USER, ""}} {
		relation, err := platformRoleRelation(test.role)
		if err != nil || relation != test.relation {
			t.Fatalf("role %s: %q %v", test.role, relation, err)
		}
	}
	if _, err := platformRoleRelation(usersv1.PlatformRole(999)); err == nil {
		t.Fatal("unknown platform role accepted")
	}
}
