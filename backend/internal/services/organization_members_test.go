package services_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/services/orgmembers"
)

func TestAddOrganizationMember_RequiresAuth(t *testing.T) {
	svc := orgmembers.NewService(nil, nil, &authz.Enforcer{}, nil)

	req := connect.NewRequest(&eventsv1.AddOrganizationMemberRequest{
		OrganizationId: 1,
		UserId:         2,
		RoleTemplateId: 1,
	})

	_, err := svc.AddOrganizationMember(context.Background(), req)
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

func TestRemoveOrganizationMemberByUserAndOrg_RequiresAuth(t *testing.T) {
	svc := orgmembers.NewService(nil, nil, &authz.Enforcer{}, nil)

	req := connect.NewRequest(&eventsv1.RemoveOrganizationMemberByUserAndOrgRequest{
		OrganizationId: 1,
		UserId:         2,
	})

	_, err := svc.RemoveOrganizationMemberByUserAndOrg(context.Background(), req)
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
