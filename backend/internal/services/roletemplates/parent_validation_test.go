package roletemplates

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestValidateParentTemplate(t *testing.T) {
	org, otherOrg := int32(10), int32(11)
	one, two, three, four, zero, negative := int32(1), int32(2), int32(3), int32(4), int32(0), int32(-1)
	local := models.RoleTemplate{ID: one, OrganizationTypeID: 5, OrganizationID: &org}
	global := models.RoleTemplate{ID: one, OrganizationTypeID: 5}
	dbFailure := errors.New("database offline")
	for _, test := range []struct {
		name      string
		child     models.RoleTemplate
		parentID  *int32
		ancestors []models.RoleTemplate
		lookupErr error
		code      connect.Code
		message   string
	}{
		{name: "no parent", child: local},
		{name: "local parent in same organization", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5, OrganizationID: &org}}},
		{name: "local child with global parent", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5}}},
		{name: "global child with global parent", child: global, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5}}},
		{name: "new child has no assigned id", child: models.RoleTemplate{OrganizationTypeID: 5, OrganizationID: &org}, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5}}},
		{name: "valid ancestor chain", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5, OrganizationID: &org, ParentTemplateID: &three}, {ID: three, OrganizationTypeID: 5, ParentTemplateID: &four}, {ID: four, OrganizationTypeID: 5}}},
		{name: "global cannot use local parent", child: global, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5, OrganizationID: &org}}, code: connect.CodeInvalidArgument, message: "scope"},
		{name: "different organization", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5, OrganizationID: &otherOrg}}, code: connect.CodeInvalidArgument, message: "scope"},
		{name: "different organization type", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 6}}, code: connect.CodeInvalidArgument, message: "organization type"},
		{name: "self parent", child: local, parentID: &one, code: connect.CodeInvalidArgument, message: "cycle"},
		{name: "descendant parent", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5, ParentTemplateID: &three}, {ID: three, OrganizationTypeID: 5, ParentTemplateID: &one}}, code: connect.CodeInvalidArgument, message: "cycle"},
		{name: "existing ancestor cycle", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5, ParentTemplateID: &three}, {ID: three, OrganizationTypeID: 5, ParentTemplateID: &two}}, code: connect.CodeInvalidArgument, message: "cycle"},
		{name: "global ancestor cannot use local parent", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5, ParentTemplateID: &three}, {ID: three, OrganizationTypeID: 5, OrganizationID: &org}}, code: connect.CodeInvalidArgument, message: "scope"},
		{name: "missing parent", child: local, parentID: &two, code: connect.CodeInvalidArgument, message: "does not exist"},
		{name: "zero parent", child: local, parentID: &zero, code: connect.CodeInvalidArgument, message: "positive"},
		{name: "negative parent", child: local, parentID: &negative, code: connect.CodeInvalidArgument, message: "positive"},
		{name: "malformed ancestor id", child: local, parentID: &two, ancestors: []models.RoleTemplate{{ID: two, OrganizationTypeID: 5, ParentTemplateID: &zero}}, code: connect.CodeInvalidArgument, message: "positive"},
		{name: "database failure", child: local, parentID: &two, lookupErr: dbFailure, code: connect.CodeInternal, message: "load parent template"},
	} {
		t.Run(test.name, func(t *testing.T) {
			lookupCount := 0
			err := validateParentTemplate(test.child, test.parentID, func(id int32) (models.RoleTemplate, error) {
				lookupCount++
				if lookupCount > len(test.ancestors)+1 {
					t.Fatal("ancestor validation did not terminate")
				}
				if test.lookupErr != nil {
					return models.RoleTemplate{}, test.lookupErr
				}
				for _, ancestor := range test.ancestors {
					if ancestor.ID == id {
						return ancestor, nil
					}
				}
				return models.RoleTemplate{}, sql.ErrNoRows
			})
			if test.message == "" {
				if err != nil {
					t.Fatalf("valid parent rejected: %v", err)
				}
				if test.parentID == nil && lookupCount != 0 {
					t.Fatal("omitted parent must not load ancestors")
				}
				return
			}
			if connect.CodeOf(err) != test.code || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("got %v, want %v containing %q", err, test.code, test.message)
			}
			if test.lookupErr != nil && !errors.Is(err, test.lookupErr) {
				t.Fatal("database failure cause was lost")
			}
		})
	}
}
