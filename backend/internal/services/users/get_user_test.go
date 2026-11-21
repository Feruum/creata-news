package users

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestGetUser_PublicResponseRedactsSensitiveFields(t *testing.T) {
	// Directly test the modelUserToPublicProto helper to verify redaction
	// without needing a full DB-backed service. We construct a User model
	// with sensitive fields populated and confirm none leak.
	u := models.User{
		ID:       42,
		Username: "public-user",
		Email:    "secret@example.com",
	}
	telegramID := int64(12345)
	u.TelegramID = &telegramID
	telegramUser := "tguser"
	u.TelegramUsername = &telegramUser

	proto := modelUserToPublicProto(u)

	if proto.Email != "" {
		t.Fatalf("expected public response to redact email, got %q", proto.Email)
	}
	if proto.TelegramId != nil {
		t.Fatal("expected public response to redact telegram id")
	}
	if proto.TelegramUsername != nil {
		t.Fatal("expected public response to redact telegram username")
	}
	if proto.PlatformRole != usersv1.PlatformRole_PLATFORM_ROLE_UNSPECIFIED && proto.PlatformRole != usersv1.PlatformRole_PLATFORM_ROLE_USER {
		t.Fatalf("expected public response to not expose platform role, got %v", proto.PlatformRole)
	}
	if proto.Id != 42 {
		t.Fatalf("expected id=42, got %d", proto.Id)
	}
	if proto.Username != "public-user" {
		t.Fatalf("expected username=public-user, got %q", proto.Username)
	}
}

func TestGetUserByEmail_RequiresPlatformStaff(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.GetUserByEmail(context.Background(), connect.NewRequest(&usersv1.GetUserByEmailRequest{Email: "x@example.com"}))
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}

func TestListUsers_RequiresPlatformStaff(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.ListUsers(context.Background(), connect.NewRequest(&usersv1.ListUsersRequest{}))
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}
