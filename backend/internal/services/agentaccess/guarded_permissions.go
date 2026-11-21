package agentaccess

import (
	"context"
	"fmt"
	"github.com/studyverse/ems-backend/internal/perms"
	"strconv"
)

type organizationPermissions interface {
	CheckOrgPermission(context.Context, int32, int32, string) bool
}

// The graph still governs direct shares and session membership. Organization
// inheritance is additionally bounded by current SQL membership and capabilities.
func (service *Service) checkPermission(ctx context.Context, userID, resourceType, resourceID, permission string) (bool, error) {
	if service.permissions == nil {
		return false, fmt.Errorf("authorization service unavailable")
	}
	allowed, err := service.permissions.CheckPermission(ctx, userID, resourceType, resourceID, permission)
	if err != nil || !allowed {
		return false, err
	}
	if resourceType == "document" {
		relationships, err := service.permissions.ReadRelationships(ctx, resourceType, resourceID)
		if err != nil {
			return false, err
		}
		return service.documentGrantCurrent(ctx, userID, permission, relationships), nil
	}
	if resourceType == "agent_session" && permission == "use_document" {
		relationships, err := service.permissions.ReadRelationships(ctx, resourceType, resourceID)
		if err != nil {
			return false, err
		}
		for _, relationship := range relationships {
			if relationship.Relation != "source_document" || relationship.SubjectType != "document" {
				continue
			}
			allowed, err := service.checkPermission(ctx, userID, "document", relationship.SubjectID, "view")
			if err != nil {
				return false, err
			}
			if allowed {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}
func (service *Service) documentGrantCurrent(ctx context.Context, userID, permission string, relationships []perms.Relationship) bool {
	orgPermission := ""
	switch permission {
	case "view":
		orgPermission = "view"
	case "edit", "delete":
		orgPermission = "manage_settings"
	case "share":
		orgPermission = "manage_members"
	default:
		return false
	}
	localUserID, err := strconv.ParseInt(userID, 10, 32)
	if err != nil || localUserID <= 0 {
		return false
	}
	for _, relationship := range relationships {
		if relationship.SubjectType == "user" && relationship.SubjectID == userID {
			if relationship.Relation == "owner" {
				return true
			}
			if relationship.Relation == "editor" && (permission == "view" || permission == "edit") {
				return true
			}
			if relationship.Relation == "viewer" && permission == "view" {
				return true
			}
		}
		if relationship.Relation == "parent_organization" && relationship.SubjectType == "organization" && service.organizations != nil {
			orgID, err := strconv.ParseInt(relationship.SubjectID, 10, 32)
			if err == nil && orgID > 0 && service.organizations.CheckOrgPermission(ctx, int32(localUserID), int32(orgID), orgPermission) {
				return true
			}
		}
	}
	return false
}
