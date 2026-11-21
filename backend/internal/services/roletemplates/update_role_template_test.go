package roletemplates

import (
	"strings"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestUpdateRoleTemplateClearFields(t *testing.T) {
	fields := (&eventsv1.UpdateRoleTemplateRequest{}).ProtoReflect().Descriptor().Fields()
	for name, number := range map[protoreflect.Name]protoreflect.FieldNumber{
		"clear_parent_template": 9,
		"clear_max_holders":     10,
	} {
		t.Run(string(name), func(t *testing.T) {
			field := fields.ByName(name)
			if field == nil {
				t.Fatalf("missing explicit clear field %s", name)
			}
			if field.Number() != number || field.Kind() != protoreflect.BoolKind || field.HasPresence() {
				t.Fatalf("%s must be a non-optional bool at field %d", name, number)
			}
		})
	}
}

func TestApplyRoleTemplateUpdate(t *testing.T) {
	parent, limit, replacement := int32(2), int32(3), int32(4)
	name, description, empty := "Updated role", "Existing description", ""
	zero, disabled := int32(0), false
	for _, test := range []*struct {
		name        string
		request     eventsv1.UpdateRoleTemplateRequest
		wantParent  *int32
		wantLimit   *int32
		wantName    string
		wantChanged bool
	}{
		{name: "old request preserves nullable fields", request: eventsv1.UpdateRoleTemplateRequest{DisplayName: &name}, wantParent: &parent, wantLimit: &limit, wantName: name, wantChanged: true},
		{name: "empty request preserves all fields", wantParent: &parent, wantLimit: &limit, wantName: "Original"},
		{name: "clear only parent", request: eventsv1.UpdateRoleTemplateRequest{ClearParentTemplate: true}, wantLimit: &limit, wantName: "Original", wantChanged: true},
		{name: "clear only limit", request: eventsv1.UpdateRoleTemplateRequest{ClearMaxHolders: true}, wantParent: &parent, wantName: "Original", wantChanged: true},
		{name: "clear both", request: eventsv1.UpdateRoleTemplateRequest{ClearParentTemplate: true, ClearMaxHolders: true}, wantName: "Original", wantChanged: true},
		{name: "set both", request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &replacement, MaxHolders: &replacement}, wantParent: &replacement, wantLimit: &replacement, wantName: "Original", wantChanged: true},
		{name: "explicit zero and false values", request: eventsv1.UpdateRoleTemplateRequest{Description: &empty, SortOrder: &zero, IsRequired: &disabled, IsActive: &disabled}, wantParent: &parent, wantLimit: &limit, wantName: "Original", wantChanged: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			template := models.RoleTemplate{ID: 1, OrganizationTypeID: 5, Key: "leader", DisplayName: "Original", Description: &description, ParentTemplateID: &parent, MaxHolders: &limit, SortOrder: 7, IsRequired: true, IsActive: true}
			columns := applyRoleTemplateUpdate(&template, &test.request)
			if (len(columns) > 0) != test.wantChanged {
				t.Fatalf("columns %v, want changed %v", columns, test.wantChanged)
			}
			assertInt32Pointer := func(name string, got, want *int32) {
				t.Helper()
				if got == nil || want == nil {
					if got != want {
						t.Errorf("%s null state differs: got %v, want %v", name, got, want)
					}
					return
				}
				if *got != *want {
					t.Errorf("%s got %d, want %d", name, *got, *want)
				}
			}
			assertInt32Pointer("parent", template.ParentTemplateID, test.wantParent)
			assertInt32Pointer("limit", template.MaxHolders, test.wantLimit)
			if template.DisplayName != test.wantName || template.ID != 1 || template.Key != "leader" || template.OrganizationTypeID != 5 {
				t.Fatalf("unrequested role fields changed: %+v", template)
			}
			if test.request.Description != nil {
				if template.Description == nil || *template.Description != empty || template.SortOrder != 0 || template.IsRequired || template.IsActive {
					t.Fatalf("explicit empty, zero, and false values were ignored: %+v", template)
				}
			} else if template.Description == nil || *template.Description != description || template.SortOrder != 7 || !template.IsRequired || !template.IsActive {
				t.Fatalf("omitted role fields changed: %+v", template)
			}
		})
	}
}

func TestValidateRoleTemplateUpdate(t *testing.T) {
	positive, zero, negative := int32(2), int32(0), int32(-1)
	name := "Updated role"
	for _, test := range []*struct {
		name    string
		request eventsv1.UpdateRoleTemplateRequest
		message string
	}{
		{name: "old request omits nullable values", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, DisplayName: &name}},
		{name: "empty update", request: eventsv1.UpdateRoleTemplateRequest{Id: 1}},
		{name: "clear parent", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, ClearParentTemplate: true}},
		{name: "clear limit", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, ClearMaxHolders: true}},
		{name: "clear both", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, ClearParentTemplate: true, ClearMaxHolders: true}},
		{name: "set both", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, ParentTemplateId: &positive, MaxHolders: &positive}},
		{name: "zero role id", request: eventsv1.UpdateRoleTemplateRequest{}, message: "id must be positive"},
		{name: "negative role id", request: eventsv1.UpdateRoleTemplateRequest{Id: negative}, message: "id must be positive"},
		{name: "zero parent", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, ParentTemplateId: &zero}, message: "parent_template_id must be positive"},
		{name: "negative parent", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, ParentTemplateId: &negative}, message: "parent_template_id must be positive"},
		{name: "zero limit", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, MaxHolders: &zero}, message: "max_holders must be positive"},
		{name: "negative limit", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, MaxHolders: &negative}, message: "max_holders must be positive"},
		{name: "parent clear conflict", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, ParentTemplateId: &positive, ClearParentTemplate: true}, message: "clear_parent_template"},
		{name: "limit clear conflict", request: eventsv1.UpdateRoleTemplateRequest{Id: 1, MaxHolders: &positive, ClearMaxHolders: true}, message: "clear_max_holders"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateRoleTemplateUpdate(&test.request)
			if test.message == "" {
				if err != nil {
					t.Fatalf("valid update rejected: %v", err)
				}
				return
			}
			if connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("got %v, want invalid_argument containing %q", err, test.message)
			}
		})
	}
}
