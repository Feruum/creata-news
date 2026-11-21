package authz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
	"strconv"
	"time"
)

func membershipEligible(member models.OrganizationMember, template models.RoleTemplate, org models.Organization, now time.Time) bool {
	return member.OrganizationID == org.ID && template.IsActive && template.OrganizationTypeID == org.OrganizationTypeID &&
		(template.OrganizationID == nil || *template.OrganizationID == org.ID) && (member.ValidUntil == nil || member.ValidUntil.After(now))
}

// The database bounds grants even when graph revocation or role synchronization is delayed.
func (e *Enforcer) orgGrantCurrentResult(ctx context.Context, userID, orgID int32, permission string) (bool, error) {
	var member models.OrganizationMember
	if err := e.db.NewSelect().Model(&member).Where("user_id = ?", userID).Where("organization_id = ?", orgID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("read current organization grant: %w", err)
	}
	var template models.RoleTemplate
	if err := e.db.NewSelect().Model(&template).Where("id = ?", member.RoleTemplateID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("read current organization grant: %w", err)
	}
	var org models.Organization
	if err := e.db.NewSelect().Model(&org).Where("id = ?", orgID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("read current organization grant: %w", err)
	}
	if !membershipEligible(member, template, org, time.Now()) {
		return false, nil
	}
	if permission == "view" {
		return true, nil
	}
	exists, err := e.db.NewSelect().Model((*models.RoleTemplateCapability)(nil)).Where("role_template_id = ?", template.ID).Where("capability = ?", permission).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("read current capabilities: %w", err)
	}
	return exists, nil
}
func (e *Enforcer) CheckResourcePermission(ctx context.Context, userID int32, resourceType, resourceID, permission string) bool {
	if e.perms == nil {
		return false
	}
	if resourceType != "event" {
		allowed, err := e.perms.CheckPermission(ctx, fmt.Sprint(userID), resourceType, resourceID, permission)
		return err == nil && allowed
	}
	eventID, err := strconv.ParseInt(resourceID, 10, 32)
	if err != nil {
		return false
	}
	var event models.Event
	if err := e.db.NewSelect().Model(&event).Where("id = ?", int32(eventID)).Scan(ctx); err != nil {
		return false
	}
	orgPermission := ""
	switch permission {
	case "edit", "delete":
		orgPermission = "create_event"
	case "manage_registrations":
		orgPermission = "manage_registrations"
	case "check_in":
		orgPermission = "check_in_attendees"
	case "view":
		orgPermission = "view"
	case "view_analytics":
		orgPermission = "view_analytics"
	default:
		return false
	}
	if e.CheckPlatformStaff(ctx, userID) {
		return true
	}
	// DB creator identity also covers events written with legacy Kratos-ID creator tuples.
	if event.UserID == userID {
		return true
	}
	if e.CheckOrgPermission(ctx, userID, event.OrganizationID, orgPermission) {
		return true
	}
	var collaborators []models.EventCollaborator
	if err := e.db.NewSelect().Model(&collaborators).Where("event_id = ?", event.ID).Scan(ctx); err != nil {
		return false
	}
	for _, collaborator := range collaborators {
		if e.CheckOrgPermission(ctx, userID, collaborator.OrganizationID, orgPermission) {
			return true
		}
	}
	return false
}

// SyncOrganizationMember reconciles current database membership to SpiceDB.
func (e *Enforcer) SyncOrganizationMember(ctx context.Context, orgID, userID int32) error {
	if e.perms == nil {
		return fmt.Errorf("authorization service unavailable")
	}
	return e.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := LockOrganizationMember(ctx, tx, orgID, userID); err != nil {
			return err
		}
		var member models.OrganizationMember
		if err := tx.NewSelect().Model(&member).Where("user_id = ?", userID).Where("organization_id = ?", orgID).Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return e.perms.RemoveOrganizationMember(ctx, fmt.Sprint(orgID), fmt.Sprint(userID))
			}
			return fmt.Errorf("load membership: %w", err)
		}
		var template models.RoleTemplate
		if err := tx.NewSelect().Model(&template).Where("id = ?", member.RoleTemplateID).Scan(ctx); err != nil {
			return fmt.Errorf("load role template: %w", err)
		}
		var org models.Organization
		if err := tx.NewSelect().Model(&org).Where("id = ?", orgID).Scan(ctx); err != nil {
			return fmt.Errorf("load organization: %w", err)
		}
		if !membershipEligible(member, template, org, time.Now()) {
			return e.perms.RemoveOrganizationMember(ctx, fmt.Sprint(orgID), fmt.Sprint(userID))
		}
		var caps []models.RoleTemplateCapability
		err := tx.NewSelect().Model(&caps).Join("JOIN organization_members AS om ON om.role_template_id = rc.role_template_id").Where("om.organization_id = ?", orgID).Where("om.user_id = ?", userID).Scan(ctx)
		if err != nil {
			return fmt.Errorf("load member capabilities: %w", err)
		}
		relations := make([]string, 0, len(caps))
		for _, cap := range caps {
			relations = append(relations, "can_"+string(cap.Capability))
		}
		return e.perms.SetOrganizationMember(ctx, fmt.Sprint(orgID), fmt.Sprint(userID), relations)
	})
}
