package organizations

import (
	"time"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

type organizationWithSlug struct {
	models.Organization
	IsCurrent     bool   `bun:"is_current"`
	CanonicalSlug string `bun:"canonical_slug"`
}

func modelOrganizationToProto(o models.Organization) *eventsv1.Organization {
	status := eventsv1.OrganizationStatus_ORGANIZATION_STATUS_ACTIVE
	if o.Status != nil {
		switch *o.Status {
		case models.OrganizationStatusArchived:
			status = eventsv1.OrganizationStatus_ORGANIZATION_STATUS_ARCHIVED
		case models.OrganizationStatusFrozen:
			status = eventsv1.OrganizationStatus_ORGANIZATION_STATUS_FROZEN
		}
	}

	org := &eventsv1.Organization{
		Id:                 o.ID,
		Title:              o.Title,
		OrganizationTypeId: o.OrganizationTypeID,
		Status:             status,
		CreatedAt:          o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          o.UpdatedAt.Format(time.RFC3339),
	}
	if o.ImageUrl != nil {
		org.ImageUrl = o.ImageUrl
	}
	if len(o.Description) > 0 {
		desc := string(o.Description)
		org.Description = &desc
	}
	if o.Instagram != nil {
		org.Instagram = o.Instagram
	}
	if o.TelegramChannel != nil {
		org.TelegramChannel = o.TelegramChannel
	}
	if o.TelegramChat != nil {
		org.TelegramChat = o.TelegramChat
	}
	if o.Website != nil {
		org.Website = o.Website
	}
	if o.Youtube != nil {
		org.Youtube = o.Youtube
	}
	if o.Tiktok != nil {
		org.Tiktok = o.Tiktok
	}
	if o.Linkedin != nil {
		org.Linkedin = o.Linkedin
	}
	return org
}

func protoStatusToDB(status eventsv1.OrganizationStatus) models.OrganizationStatus {
	switch status {
	case eventsv1.OrganizationStatus_ORGANIZATION_STATUS_ARCHIVED:
		return models.OrganizationStatusArchived
	case eventsv1.OrganizationStatus_ORGANIZATION_STATUS_FROZEN:
		return models.OrganizationStatusFrozen
	default:
		return models.OrganizationStatusActive
	}
}

// protoStatusToString converts a proto organization status to a string
// nolint:unused
func protoStatusToString(status eventsv1.OrganizationStatus) string {
	switch status {
	case eventsv1.OrganizationStatus_ORGANIZATION_STATUS_ARCHIVED:
		return "archived"
	case eventsv1.OrganizationStatus_ORGANIZATION_STATUS_FROZEN:
		return "frozen"
	default:
		return "active"
	}
}

func organizationRowToOrganization(row organizationWithSlug) models.Organization {
	return row.Organization
}
