package roletemplates

import (
	"time"

	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func dbRoleTemplateToProto(t models.RoleTemplate, caps []models.RoleTemplateCapability) *eventsv1.RoleTemplate {
	proto := &eventsv1.RoleTemplate{
		Id:                 t.ID,
		OrganizationTypeId: t.OrganizationTypeID,
		Key:                t.Key,
		DisplayName:        t.DisplayName,
		SortOrder:          t.SortOrder,
		IsRequired:         t.IsRequired,
		IsActive:           t.IsActive,
		CreatedAt:          t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          t.UpdatedAt.Format(time.RFC3339),
	}

	if t.OrganizationID != nil {
		proto.OrganizationId = t.OrganizationID
	}
	if t.Description != nil {
		proto.Description = t.Description
	}
	if t.ParentTemplateID != nil {
		proto.ParentTemplateId = t.ParentTemplateID
	}
	if t.MaxHolders != nil {
		proto.MaxHolders = t.MaxHolders
	}

	protoCaps := lo.Map(caps, func(c models.RoleTemplateCapability, _ int) eventsv1.Capability {
		return dbCapabilityToProto(c)
	})
	proto.Capabilities = protoCaps

	return proto
}

func dbCapabilityToProto(c models.RoleTemplateCapability) eventsv1.Capability {
	switch c.Capability {
	case models.CapabilityCreateEvent:
		return eventsv1.Capability_CAPABILITY_CREATE_EVENT
	case models.CapabilityManageMembers:
		return eventsv1.Capability_CAPABILITY_MANAGE_MEMBERS
	case models.CapabilityManageSettings:
		return eventsv1.Capability_CAPABILITY_MANAGE_SETTINGS
	case models.CapabilityViewAnalytics:
		return eventsv1.Capability_CAPABILITY_VIEW_ANALYTICS
	case models.CapabilityBookRooms:
		return eventsv1.Capability_CAPABILITY_BOOK_ROOMS
	case models.CapabilityApproveBookings:
		return eventsv1.Capability_CAPABILITY_APPROVE_BOOKINGS
	case models.CapabilityManageRegistrations:
		return eventsv1.Capability_CAPABILITY_MANAGE_REGISTRATIONS
	case models.CapabilityCheckInAttendees:
		return eventsv1.Capability_CAPABILITY_CHECK_IN_ATTENDEES
	default:
		return eventsv1.Capability_CAPABILITY_UNSPECIFIED
	}
}

func protoCapabilityToDB(c eventsv1.Capability) models.Capability {
	switch c {
	case eventsv1.Capability_CAPABILITY_CREATE_EVENT:
		return models.CapabilityCreateEvent
	case eventsv1.Capability_CAPABILITY_MANAGE_MEMBERS:
		return models.CapabilityManageMembers
	case eventsv1.Capability_CAPABILITY_MANAGE_SETTINGS:
		return models.CapabilityManageSettings
	case eventsv1.Capability_CAPABILITY_VIEW_ANALYTICS:
		return models.CapabilityViewAnalytics
	case eventsv1.Capability_CAPABILITY_BOOK_ROOMS:
		return models.CapabilityBookRooms
	case eventsv1.Capability_CAPABILITY_APPROVE_BOOKINGS:
		return models.CapabilityApproveBookings
	case eventsv1.Capability_CAPABILITY_MANAGE_REGISTRATIONS:
		return models.CapabilityManageRegistrations
	case eventsv1.Capability_CAPABILITY_CHECK_IN_ATTENDEES:
		return models.CapabilityCheckInAttendees
	default:
		return ""
	}
}
