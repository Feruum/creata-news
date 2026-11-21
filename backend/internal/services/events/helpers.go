package events

import (
	"encoding/json"
	"log/slog"
	"time"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
)

func dbEventToProto(e models.Event, org *models.Organization, category *models.Category) *eventsv1.Event {
	format := eventsv1.EventFormat_EVENT_FORMAT_OFFLINE
	if e.Format != nil {
		switch *e.Format {
		case models.FormatOnline:
			format = eventsv1.EventFormat_EVENT_FORMAT_ONLINE
		case models.FormatHybrid:
			format = eventsv1.EventFormat_EVENT_FORMAT_HYBRID
		}
	}

	graceMinutes := int32(120)
	if e.CheckInGraceMinutes != nil {
		graceMinutes = *e.CheckInGraceMinutes
	}

	event := &eventsv1.Event{
		Id:                  e.ID,
		PublishOnMap:        e.PublishOnMap,
		PublishOnSundayOpen: e.PublishOnSundayOpen,
		Title:               e.Title,
		Description:         string(e.Description),
		UserId:              e.UserID,
		OrganizationId:      e.OrganizationID,
		StartTime:           e.StartTime.Format(time.RFC3339),
		EndTime:             e.EndTime.Format(time.RFC3339),
		Format:              format,
		CreatedAt:           e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           e.UpdatedAt.Format(time.RFC3339),
		Status:              eventStatusToProto(e.Status),
		CheckInPhase:        checkInPhaseToProto(models.DeriveCheckInPhase(e.Status, time.Now(), e.StartTime, e.EndTime, int(graceMinutes))),
		CheckInGraceMinutes: &graceMinutes,
		SocGpaBadge:         dbBadgeToProto(e.SocGpaBadgeStatus, e.SocGpaBadgeReviewedBy, e.SocGpaBadgeReviewedAt),
		IrosBadge:           dbBadgeToProto(e.IrosBadgeStatus, e.IrosBadgeReviewedBy, e.IrosBadgeReviewedAt),
	}
	if e.ImageUrl != nil {
		event.ImageUrl = e.ImageUrl
	}
	if e.CategoryID != nil {
		event.CategoryId = e.CategoryID
	}
	if e.Capacity != nil {
		event.Capacity = e.Capacity
	}
	if org != nil {
		event.Organization = dbOrganizationToProto(*org)
	}
	if category != nil {
		event.Category = dbCategoryToProto(category)
	}
	if len(e.StructuredLocation) > 0 {
		event.StructuredLocation = parseStructuredLocation(e.StructuredLocation)
	}
	if e.CompletedAt != nil {
		v := e.CompletedAt.Format(time.RFC3339)
		event.CompletedAt = &v
	}
	if e.PublishedAt != nil {
		v := e.PublishedAt.Format(time.RFC3339)
		event.PublishedAt = &v
	}
	if e.CancelledAt != nil {
		v := e.CancelledAt.Format(time.RFC3339)
		event.CancelledAt = &v
	}
	if e.CancelledBy != nil {
		event.CancelledBy = e.CancelledBy
	}
	if e.CancelReason != nil {
		event.CancelReason = e.CancelReason
	}
	if e.CheckInToken != nil {
		event.CheckInToken = e.CheckInToken
	}

	return event
}

func dbEventWithRelationsToProto(e models.Event, org models.Organization, category *models.Category) *eventsv1.Event {
	return dbEventToProto(e, &org, category)
}

func dbOrganizationToProto(o models.Organization) *eventsv1.Organization {
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

func dbCategoryToProto(c *models.Category) *eventsv1.Category {
	return &eventsv1.Category{
		Id:        c.ID,
		Name:      c.Name,
		Color:     c.Color,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type structuredLocationJSON struct {
	Type            string   `json:"type"`
	RoomID          *string  `json:"room_id,omitempty"`
	RoomName        *string  `json:"room_name,omitempty"`
	Floor           *int32   `json:"floor,omitempty"`
	ExternalAddress *string  `json:"external_address,omitempty"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	GoogleMapsUrl   *string  `json:"google_maps_url,omitempty"`
	OnlineUrl       *string  `json:"online_url,omitempty"`
	OnlinePlatform  *string  `json:"online_platform,omitempty"`
}

func parseStructuredLocation(data []byte) *eventsv1.EventLocation {
	if len(data) == 0 {
		return nil
	}
	var loc structuredLocationJSON
	if err := json.Unmarshal(data, &loc); err != nil {
		slog.Warn("Failed to parse structured_location", "error", err)
		return nil
	}

	locationType := eventsv1.LocationType_LOCATION_TYPE_UNSPECIFIED
	switch loc.Type {
	case "campus_room":
		locationType = eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM
	case "external":
		locationType = eventsv1.LocationType_LOCATION_TYPE_EXTERNAL
	case "online":
		locationType = eventsv1.LocationType_LOCATION_TYPE_ONLINE
	case "hybrid":
		locationType = eventsv1.LocationType_LOCATION_TYPE_HYBRID
	}

	result := &eventsv1.EventLocation{Type: locationType}
	if loc.RoomID != nil {
		normalizedRoomID := roomid.Normalize(*loc.RoomID)
		if normalizedRoomID != "" {
			result.RoomId = &normalizedRoomID
		}
	}
	if loc.RoomName != nil {
		result.RoomName = loc.RoomName
	}
	if loc.Floor != nil {
		result.Floor = loc.Floor
	}
	if loc.ExternalAddress != nil {
		result.ExternalAddress = loc.ExternalAddress
	}
	if loc.Latitude != nil {
		result.Latitude = loc.Latitude
	}
	if loc.Longitude != nil {
		result.Longitude = loc.Longitude
	}
	if loc.GoogleMapsUrl != nil {
		result.GoogleMapsUrl = loc.GoogleMapsUrl
	}
	if loc.OnlineUrl != nil {
		result.OnlineUrl = loc.OnlineUrl
	}
	if loc.OnlinePlatform != nil {
		result.OnlinePlatform = loc.OnlinePlatform
	}
	return result
}

func serializeStructuredLocation(loc *eventsv1.EventLocation) []byte {
	if loc == nil {
		return nil
	}

	locType := "unspecified"
	switch loc.Type {
	case eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM:
		locType = "campus_room"
	case eventsv1.LocationType_LOCATION_TYPE_EXTERNAL:
		locType = "external"
	case eventsv1.LocationType_LOCATION_TYPE_ONLINE:
		locType = "online"
	case eventsv1.LocationType_LOCATION_TYPE_HYBRID:
		locType = "hybrid"
	}

	jsonLoc := structuredLocationJSON{
		Type:            locType,
		RoomName:        loc.RoomName,
		Floor:           loc.Floor,
		ExternalAddress: loc.ExternalAddress,
		Latitude:        loc.Latitude,
		Longitude:       loc.Longitude,
		GoogleMapsUrl:   loc.GoogleMapsUrl,
		OnlineUrl:       loc.OnlineUrl,
		OnlinePlatform:  loc.OnlinePlatform,
	}
	if loc.RoomId != nil {
		normalizedRoomID := roomid.Normalize(*loc.RoomId)
		if normalizedRoomID != "" {
			jsonLoc.RoomID = &normalizedRoomID
		}
	}

	data, err := json.Marshal(jsonLoc)
	if err != nil {
		slog.Error("Failed to serialize structured_location", "error", err)
		return nil
	}
	return data
}

func eventStatusToProto(status models.EventStatus) eventsv1.EventStatus {
	switch status {
	case models.EventStatusDraft:
		return eventsv1.EventStatus_EVENT_STATUS_DRAFT
	case models.EventStatusPublished:
		return eventsv1.EventStatus_EVENT_STATUS_PUBLISHED
	case models.EventStatusArchived:
		return eventsv1.EventStatus_EVENT_STATUS_ARCHIVED
	case models.EventStatusCancelled:
		return eventsv1.EventStatus_EVENT_STATUS_CANCELLED
	default:
		return eventsv1.EventStatus_EVENT_STATUS_UNSPECIFIED
	}
}

func checkInPhaseToProto(phase models.CheckInPhase) eventsv1.CheckInPhase {
	switch phase {
	case models.CheckInPhaseNone:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_NONE
	case models.CheckInPhaseLive:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_LIVE
	case models.CheckInPhaseGrace:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_GRACE
	case models.CheckInPhaseLocked:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_LOCKED
	default:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_UNSPECIFIED
	}
}

func protoEventStatusToModel(status eventsv1.EventStatus) models.EventStatus {
	switch status {
	case eventsv1.EventStatus_EVENT_STATUS_DRAFT:
		return models.EventStatusDraft
	case eventsv1.EventStatus_EVENT_STATUS_PUBLISHED:
		return models.EventStatusPublished
	case eventsv1.EventStatus_EVENT_STATUS_ARCHIVED:
		return models.EventStatusArchived
	case eventsv1.EventStatus_EVENT_STATUS_CANCELLED:
		return models.EventStatusCancelled
	default:
		return models.EventStatusPublished
	}
}

func deriveFormatFromLocationType(locType eventsv1.LocationType) *models.Format {
	switch locType {
	case eventsv1.LocationType_LOCATION_TYPE_ONLINE:
		f := models.FormatOnline
		return &f
	case eventsv1.LocationType_LOCATION_TYPE_HYBRID:
		f := models.FormatHybrid
		return &f
	default:
		f := models.FormatOffline
		return &f
	}
}

type eventCollaboratorRow struct {
	models.EventCollaborator
	OrganizationTitle    *string `bun:"organization_title"`
	OrganizationImageUrl *string `bun:"organization_image_url"`
	OrganizationSlug     *string `bun:"organization_slug"`
}

func dbEventCollaboratorRowToProto(row eventCollaboratorRow) *eventsv1.EventCollaborator {
	proto := &eventsv1.EventCollaborator{
		Id:             row.ID,
		EventId:        row.EventID,
		OrganizationId: row.OrganizationID,
		AddedAt:        row.AddedAt.Format(time.RFC3339),
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
	}
	if row.AddedBy != nil {
		proto.AddedBy = row.AddedBy
	}
	proto.OrganizationTitle = row.OrganizationTitle
	if row.OrganizationImageUrl != nil {
		proto.OrganizationImageUrl = row.OrganizationImageUrl
	}
	if row.OrganizationSlug != nil {
		proto.OrganizationSlug = row.OrganizationSlug
	}
	return proto
}

type eventOrganizerRow struct {
	models.EventOrganizer
	Username          string  `bun:"username"`
	Email             string  `bun:"email"`
	FirstName         *string `bun:"first_name"`
	LastName          *string `bun:"last_name"`
	OrganizationTitle string  `bun:"organization_title"`
}

func dbEventOrganizerRowToProto(row eventOrganizerRow) *eventsv1.EventOrganizer {
	proto := &eventsv1.EventOrganizer{
		Id:             row.ID,
		EventId:        row.EventID,
		UserId:         row.UserID,
		OrganizationId: row.OrganizationID,
		AddedAt:        row.AddedAt.Format(time.RFC3339),
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
	}
	if row.Role != nil {
		proto.Role = row.Role
	}
	if row.AddedBy != nil {
		proto.AddedBy = row.AddedBy
	}
	proto.Username = &row.Username
	proto.Email = &row.Email
	proto.FirstName = row.FirstName
	proto.LastName = row.LastName
	proto.OrganizationTitle = &row.OrganizationTitle
	return proto
}

func dbInviteStatusToProto(status models.CollaborationInviteStatus) eventsv1.CollaborationInviteStatus {
	switch status {
	case models.CollaborationInviteStatusPending:
		return eventsv1.CollaborationInviteStatus_COLLABORATION_INVITE_STATUS_PENDING
	case models.CollaborationInviteStatusAccepted:
		return eventsv1.CollaborationInviteStatus_COLLABORATION_INVITE_STATUS_ACCEPTED
	case models.CollaborationInviteStatusDeclined:
		return eventsv1.CollaborationInviteStatus_COLLABORATION_INVITE_STATUS_DECLINED
	case models.CollaborationInviteStatusRevoked:
		return eventsv1.CollaborationInviteStatus_COLLABORATION_INVITE_STATUS_REVOKED
	case models.CollaborationInviteStatusExpired:
		return eventsv1.CollaborationInviteStatus_COLLABORATION_INVITE_STATUS_EXPIRED
	default:
		return eventsv1.CollaborationInviteStatus_COLLABORATION_INVITE_STATUS_UNSPECIFIED
	}
}

func dbCollaborationInviteToProto(invite models.EventCollaborationInvite) *eventsv1.CollaborationInvite {
	proto := &eventsv1.CollaborationInvite{
		Id:              invite.ID,
		EventId:         invite.EventID,
		OrganizationId:  invite.OrganizationID,
		InvitedByUserId: invite.InvitedByUserID,
		Status:          dbInviteStatusToProto(invite.Status),
		Token:           invite.Token,
		ExpiresAt:       invite.ExpiresAt.Format(time.RFC3339),
		CreatedAt:       invite.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       invite.UpdatedAt.Format(time.RFC3339),
	}
	if invite.Message != nil {
		proto.Message = invite.Message
	}
	if invite.RespondedAt != nil {
		value := invite.RespondedAt.Format(time.RFC3339)
		proto.RespondedAt = &value
	}
	if invite.RespondedByUserID != nil {
		proto.RespondedByUserId = invite.RespondedByUserID
	}
	return proto
}

type collaborationInviteByEventRow struct {
	models.EventCollaborationInvite
	OrganizationTitle    *string `bun:"organization_title"`
	OrganizationImageUrl *string `bun:"organization_image_url"`
	InvitedByUsername    string  `bun:"invited_by_username"`
	InvitedByFirstName   *string `bun:"invited_by_first_name"`
	InvitedByLastName    *string `bun:"invited_by_last_name"`
}

func dbCollaborationInviteByEventRowToProto(row collaborationInviteByEventRow) *eventsv1.CollaborationInvite {
	proto := &eventsv1.CollaborationInvite{
		Id:              row.ID,
		EventId:         row.EventID,
		OrganizationId:  row.OrganizationID,
		InvitedByUserId: row.InvitedByUserID,
		Status:          dbInviteStatusToProto(row.Status),
		Token:           row.Token,
		ExpiresAt:       row.ExpiresAt.Format(time.RFC3339),
		CreatedAt:       row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       row.UpdatedAt.Format(time.RFC3339),
	}
	if row.Message != nil {
		proto.Message = row.Message
	}
	if row.RespondedAt != nil {
		value := row.RespondedAt.Format(time.RFC3339)
		proto.RespondedAt = &value
	}
	if row.RespondedByUserID != nil {
		proto.RespondedByUserId = row.RespondedByUserID
	}
	proto.OrganizationTitle = row.OrganizationTitle
	if row.OrganizationImageUrl != nil {
		proto.OrganizationImageUrl = row.OrganizationImageUrl
	}
	proto.InvitedByUsername = &row.InvitedByUsername
	proto.InvitedByFirstName = row.InvitedByFirstName
	proto.InvitedByLastName = row.InvitedByLastName
	return proto
}

type pendingInviteRow struct {
	models.EventCollaborationInvite
	EventTitle               string    `bun:"event_title"`
	EventSlug                *string   `bun:"event_slug"`
	StartTime                time.Time `bun:"start_time"`
	EndTime                  time.Time `bun:"end_time"`
	HostOrganizationTitle    string    `bun:"host_organization_title"`
	HostOrganizationImageUrl *string   `bun:"host_organization_image_url"`
	InvitedByUsername        string    `bun:"invited_by_username"`
	InvitedByFirstName       *string   `bun:"invited_by_first_name"`
	InvitedByLastName        *string   `bun:"invited_by_last_name"`
}

func dbCollaborationInvitePendingRowToProto(row pendingInviteRow) *eventsv1.CollaborationInvite {
	proto := &eventsv1.CollaborationInvite{
		Id:              row.ID,
		EventId:         row.EventID,
		OrganizationId:  row.OrganizationID,
		InvitedByUserId: row.InvitedByUserID,
		Status:          dbInviteStatusToProto(row.Status),
		Token:           row.Token,
		ExpiresAt:       row.ExpiresAt.Format(time.RFC3339),
		CreatedAt:       row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       row.UpdatedAt.Format(time.RFC3339),
	}
	if row.Message != nil {
		proto.Message = row.Message
	}
	if row.RespondedAt != nil {
		value := row.RespondedAt.Format(time.RFC3339)
		proto.RespondedAt = &value
	}
	if row.RespondedByUserID != nil {
		proto.RespondedByUserId = row.RespondedByUserID
	}
	proto.EventTitle = &row.EventTitle
	startTime := row.StartTime.Format(time.RFC3339)
	endTime := row.EndTime.Format(time.RFC3339)
	proto.EventStartTime = &startTime
	proto.EventEndTime = &endTime
	proto.HostOrganizationTitle = &row.HostOrganizationTitle
	if row.HostOrganizationImageUrl != nil {
		proto.HostOrganizationImageUrl = row.HostOrganizationImageUrl
	}
	proto.InvitedByUsername = &row.InvitedByUsername
	proto.InvitedByFirstName = row.InvitedByFirstName
	proto.InvitedByLastName = row.InvitedByLastName
	proto.EventSlug = row.EventSlug
	return proto
}

type inviteByTokenRow struct {
	models.EventCollaborationInvite
	EventTitle        string  `bun:"event_title"`
	EventSlug         *string `bun:"event_slug"`
	OrganizationTitle string  `bun:"organization_title"`
}

func dbCollaborationInviteByTokenRowToProto(row inviteByTokenRow) *eventsv1.CollaborationInvite {
	proto := &eventsv1.CollaborationInvite{
		Id:              row.ID,
		EventId:         row.EventID,
		OrganizationId:  row.OrganizationID,
		InvitedByUserId: row.InvitedByUserID,
		Status:          dbInviteStatusToProto(row.Status),
		ExpiresAt:       row.ExpiresAt.Format(time.RFC3339),
		CreatedAt:       row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       row.UpdatedAt.Format(time.RFC3339),
	}
	if row.Message != nil {
		proto.Message = row.Message
	}
	if row.RespondedAt != nil {
		value := row.RespondedAt.Format(time.RFC3339)
		proto.RespondedAt = &value
	}
	if row.RespondedByUserID != nil {
		proto.RespondedByUserId = row.RespondedByUserID
	}
	proto.EventTitle = &row.EventTitle
	proto.EventSlug = row.EventSlug
	proto.OrganizationTitle = &row.OrganizationTitle
	return proto
}

type eventSlugRow struct {
	models.Event
	Slug          string `bun:"slug"`
	IsCurrent     bool   `bun:"is_current"`
	CanonicalSlug string `bun:"canonical_slug"`
}
