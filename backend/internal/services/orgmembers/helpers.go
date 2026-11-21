package orgmembers

import (
	"encoding/json"
	"time"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

// organizationMemberRow is a custom struct for joined queries
type organizationMemberRow struct {
	bun.BaseModel `bun:"table:organization_members,alias:om"`
	models.OrganizationMember
	UserUsername     string  `bun:"user_username"`
	UserEmail        string  `bun:"user_email"`
	UserFirstName    *string `bun:"user_first_name"`
	UserLastName     *string `bun:"user_last_name"`
	RoleTemplateName string  `bun:"role_template_name"`
	RoleTemplateKey  string  `bun:"role_template_key"`
}

func dbOrganizationMemberToProto(m organizationMemberRow) *eventsv1.OrganizationMember {
	username := m.UserUsername
	email := m.UserEmail
	roleName := m.RoleTemplateName
	roleKey := m.RoleTemplateKey

	member := &eventsv1.OrganizationMember{
		Id:              m.ID,
		OrganizationId:  m.OrganizationID,
		UserId:          m.UserID,
		RoleTemplateId:  m.RoleTemplateID,
		AssignedAt:      m.AssignedAt.Format(time.RFC3339),
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       m.UpdatedAt.Format(time.RFC3339),
		Username:        &username,
		Email:           &email,
		RoleDisplayName: &roleName,
		RoleKey:         &roleKey,
	}
	if m.AssignedBy != nil {
		member.AssignedBy = m.AssignedBy
	}
	if m.ValidUntil != nil {
		validUntil := m.ValidUntil.Format(time.RFC3339)
		member.ValidUntil = &validUntil
	}
	if m.UserFirstName != nil {
		member.FirstName = m.UserFirstName
	}
	if m.UserLastName != nil {
		member.LastName = m.UserLastName
	}
	return member
}

type userOrgRow struct {
	OrganizationID     int32                      `bun:"organization_id"`
	OrganizationTitle  string                     `bun:"organization_title"`
	OrganizationSlug   string                     `bun:"organization_slug"`
	OrganizationTypeID int32                      `bun:"organization_type_id"`
	Status             *models.OrganizationStatus `bun:"status,type:organization_status"`
	CreatedAt          time.Time                  `bun:"created_at"`
	UpdatedAt          time.Time                  `bun:"updated_at"`
	ImageURL           *string                    `bun:"image_url"`
	Description        json.RawMessage            `bun:"description"`
	Instagram          *string                    `bun:"instagram"`
	TelegramChannel    *string                    `bun:"telegram_channel"`
	TelegramChat       *string                    `bun:"telegram_chat"`
	Website            *string                    `bun:"website"`
	Youtube            *string                    `bun:"youtube"`
	Tiktok             *string                    `bun:"tiktok"`
	Linkedin           *string                    `bun:"linkedin"`
	RoleTemplateID     int32                      `bun:"role_template_id"`
	RoleKey            string                     `bun:"role_key"`
	RoleDisplayName    string                     `bun:"role_display_name"`
}

func dbUserOrgRowToProto(o userOrgRow) *eventsv1.Organization {
	proto := &eventsv1.Organization{
		Id:                 o.OrganizationID,
		Title:              o.OrganizationTitle,
		Slug:               o.OrganizationSlug,
		OrganizationTypeId: o.OrganizationTypeID,
		Status:             dbStatusToProto(o.Status),
		CreatedAt:          o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          o.UpdatedAt.Format(time.RFC3339),
	}

	if o.ImageURL != nil {
		proto.ImageUrl = o.ImageURL
	}
	if len(o.Description) > 0 {
		desc := string(o.Description)
		proto.Description = &desc
	}
	if o.Instagram != nil {
		proto.Instagram = o.Instagram
	}
	if o.TelegramChannel != nil {
		proto.TelegramChannel = o.TelegramChannel
	}
	if o.TelegramChat != nil {
		proto.TelegramChat = o.TelegramChat
	}
	if o.Website != nil {
		proto.Website = o.Website
	}
	if o.Youtube != nil {
		proto.Youtube = o.Youtube
	}
	if o.Tiktok != nil {
		proto.Tiktok = o.Tiktok
	}
	if o.Linkedin != nil {
		proto.Linkedin = o.Linkedin
	}

	return proto
}

func dbStatusToProto(status *models.OrganizationStatus) eventsv1.OrganizationStatus {
	if status == nil {
		return eventsv1.OrganizationStatus_ORGANIZATION_STATUS_UNSPECIFIED
	}
	switch *status {
	case models.OrganizationStatusActive:
		return eventsv1.OrganizationStatus_ORGANIZATION_STATUS_ACTIVE
	case models.OrganizationStatusArchived:
		return eventsv1.OrganizationStatus_ORGANIZATION_STATUS_ARCHIVED
	case models.OrganizationStatusFrozen:
		return eventsv1.OrganizationStatus_ORGANIZATION_STATUS_FROZEN
	default:
		return eventsv1.OrganizationStatus_ORGANIZATION_STATUS_UNSPECIFIED
	}
}

func DBCapabilityToProto(c models.Capability) eventsv1.Capability {
	switch c {
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
