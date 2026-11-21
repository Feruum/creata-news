package users

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/authz"
)

func TestPreRegisterUser_RequiresPlatformStaff(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.PreRegisterUser(context.Background(), connect.NewRequest(&usersv1.PreRegisterUserRequest{
		Email:        "new@example.com",
		PlatformRole: usersv1.PlatformRole_PLATFORM_ROLE_STAFF,
	}))
	if err == nil {
		t.Fatal("expected unauthenticated or permission error")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}

func TestListPreRegisteredUsers_RequiresPlatformStaff(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.ListPreRegisteredUsers(context.Background(), connect.NewRequest(&usersv1.ListPreRegisteredUsersRequest{}))
	if err == nil {
		t.Fatal("expected unauthenticated or permission error")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}

func TestDeletePreRegisteredUser_RequiresPlatformStaff(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.DeletePreRegisteredUser(context.Background(), connect.NewRequest(&usersv1.DeletePreRegisteredUserRequest{
		Id: 1,
	}))
	if err == nil {
		t.Fatal("expected unauthenticated or permission error")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}
