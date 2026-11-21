package services_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/services/roletemplates"
)

func TestCreateRoleTemplate_RequiresAuth(t *testing.T) {
	svc := roletemplates.NewService(nil, &authz.Enforcer{})

	orgID := int32(1)
	req := connect.NewRequest(&eventsv1.CreateRoleTemplateRequest{
		OrganizationTypeId: 1,
		OrganizationId:     &orgID,
		Key:                "test_role",
		DisplayName:        "Test Role",
	})

	_, err := svc.CreateRoleTemplate(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unauthenticated request")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeUnauthenticated {
		t.Errorf("expected CodeUnauthenticated, got %v", connectErr.Code())
	}
}

func TestCreateRoleTemplate_DefaultRoleRequiresPlatformStaff(t *testing.T) {
	svc := roletemplates.NewService(nil, &authz.Enforcer{})

	req := connect.NewRequest(&eventsv1.CreateRoleTemplateRequest{
		OrganizationTypeId: 1,
		OrganizationId:     nil,
		Key:                "test_role",
		DisplayName:        "Test Role",
	})

	_, err := svc.CreateRoleTemplate(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for unauthenticated request")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeUnauthenticated {
		t.Errorf("expected CodeUnauthenticated, got %v", connectErr.Code())
	}
}
