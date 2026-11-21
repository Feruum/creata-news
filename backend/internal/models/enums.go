package models

import (
	"database/sql/driver"
	"fmt"
)

type AttendanceStatus string

const (
	AttendanceStatusAttended  AttendanceStatus = "attended"
	AttendanceStatusNoShow    AttendanceStatus = "no_show"
	AttendanceStatusCheckedIn AttendanceStatus = "checked_in"
)

func (e *AttendanceStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = AttendanceStatus(s)
	case string:
		*e = AttendanceStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for AttendanceStatus: %T", src)
	}
	return nil
}

func (e AttendanceStatus) Value() (driver.Value, error) {
	return string(e), nil
}

type BookingPriority string

const (
	BookingPriorityAdministrative BookingPriority = "administrative"
	BookingPriorityAcademic       BookingPriority = "academic"
	BookingPriorityStudentClub    BookingPriority = "student_club"
	BookingPriorityIndividual     BookingPriority = "individual"
)

func (e *BookingPriority) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = BookingPriority(s)
	case string:
		*e = BookingPriority(s)
	default:
		return fmt.Errorf("unsupported scan type for BookingPriority: %T", src)
	}
	return nil
}

func (e BookingPriority) Value() (driver.Value, error) {
	return string(e), nil
}

type BookingRequestStatus string

const (
	BookingRequestStatusPending   BookingRequestStatus = "pending"
	BookingRequestStatusApproved  BookingRequestStatus = "approved"
	BookingRequestStatusRejected  BookingRequestStatus = "rejected"
	BookingRequestStatusCancelled BookingRequestStatus = "cancelled"
)

func (e *BookingRequestStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = BookingRequestStatus(s)
	case string:
		*e = BookingRequestStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for BookingRequestStatus: %T", src)
	}
	return nil
}

func (e BookingRequestStatus) Value() (driver.Value, error) {
	return string(e), nil
}

type Capability string

const (
	CapabilityCreateEvent         Capability = "create_event"
	CapabilityManageMembers       Capability = "manage_members"
	CapabilityManageSettings      Capability = "manage_settings"
	CapabilityViewAnalytics       Capability = "view_analytics"
	CapabilityBookRooms           Capability = "book_rooms"
	CapabilityApproveBookings     Capability = "approve_bookings"
	CapabilityManageRegistrations Capability = "manage_registrations"
	CapabilityCheckInAttendees    Capability = "check_in_attendees"
)

func (e *Capability) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = Capability(s)
	case string:
		*e = Capability(s)
	default:
		return fmt.Errorf("unsupported scan type for Capability: %T", src)
	}
	return nil
}

func (e Capability) Value() (driver.Value, error) {
	return string(e), nil
}

type CollaborationInviteStatus string

const (
	CollaborationInviteStatusPending  CollaborationInviteStatus = "pending"
	CollaborationInviteStatusAccepted CollaborationInviteStatus = "accepted"
	CollaborationInviteStatusDeclined CollaborationInviteStatus = "declined"
	CollaborationInviteStatusRevoked  CollaborationInviteStatus = "revoked"
	CollaborationInviteStatusExpired  CollaborationInviteStatus = "expired"
)

func (e *CollaborationInviteStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CollaborationInviteStatus(s)
	case string:
		*e = CollaborationInviteStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for CollaborationInviteStatus: %T", src)
	}
	return nil
}

func (e CollaborationInviteStatus) Value() (driver.Value, error) {
	return string(e), nil
}

type Format string

const (
	FormatOnline  Format = "online"
	FormatOffline Format = "offline"
	FormatHybrid  Format = "hybrid"
)

func (e *Format) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = Format(s)
	case string:
		*e = Format(s)
	default:
		return fmt.Errorf("unsupported scan type for Format: %T", src)
	}
	return nil
}

func (e Format) Value() (driver.Value, error) {
	return string(e), nil
}

type LocationType string

const (
	LocationTypeUnspecified LocationType = "unspecified"
	LocationTypeCampusRoom  LocationType = "campus_room"
	LocationTypeExternal    LocationType = "external"
	LocationTypeOnline      LocationType = "online"
	LocationTypeHybrid      LocationType = "hybrid"
)

func (e *LocationType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = LocationType(s)
	case string:
		*e = LocationType(s)
	default:
		return fmt.Errorf("unsupported scan type for LocationType: %T", src)
	}
	return nil
}

func (e LocationType) Value() (driver.Value, error) {
	return string(e), nil
}

type OrganizationStatus string

const (
	OrganizationStatusActive   OrganizationStatus = "active"
	OrganizationStatusArchived OrganizationStatus = "archived"
	OrganizationStatusFrozen   OrganizationStatus = "frozen"
)

func (e *OrganizationStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = OrganizationStatus(s)
	case string:
		*e = OrganizationStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for OrganizationStatus: %T", src)
	}
	return nil
}

func (e OrganizationStatus) Value() (driver.Value, error) {
	return string(e), nil
}

type PlatformRole string

const (
	PlatformRoleAdmin PlatformRole = "admin"
	PlatformRoleStaff PlatformRole = "staff"
)

func (e *PlatformRole) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = PlatformRole(s)
	case string:
		*e = PlatformRole(s)
	default:
		return fmt.Errorf("unsupported scan type for PlatformRole: %T", src)
	}
	return nil
}

func (e PlatformRole) Value() (driver.Value, error) {
	return string(e), nil
}

type ExternalRegistrationStatus string

const (
	ExternalRegistrationStatusRegistered      ExternalRegistrationStatus = "registered"
	ExternalRegistrationStatusCancelled       ExternalRegistrationStatus = "cancelled"
	ExternalRegistrationStatusPendingApproval ExternalRegistrationStatus = "pending_approval"
	ExternalRegistrationStatusRejected        ExternalRegistrationStatus = "rejected"
)

func (e *ExternalRegistrationStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = ExternalRegistrationStatus(s)
	case string:
		*e = ExternalRegistrationStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for ExternalRegistrationStatus: %T", src)
	}
	return nil
}

func (e ExternalRegistrationStatus) Value() (driver.Value, error) {
	return string(e), nil
}

type RegistrationStatus string

const (
	RegistrationStatusRegistered      RegistrationStatus = "registered"
	RegistrationStatusCancelled       RegistrationStatus = "cancelled"
	RegistrationStatusWaitlist        RegistrationStatus = "waitlist"
	RegistrationStatusPendingApproval RegistrationStatus = "pending_approval"
	RegistrationStatusRejected        RegistrationStatus = "rejected"
)

func (e *RegistrationStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = RegistrationStatus(s)
	case string:
		*e = RegistrationStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for RegistrationStatus: %T", src)
	}
	return nil
}

func (e RegistrationStatus) Value() (driver.Value, error) {
	return string(e), nil
}

type EventStatus string

const (
	EventStatusDraft     EventStatus = "draft"
	EventStatusPublished EventStatus = "published"
	EventStatusArchived  EventStatus = "archived"
	EventStatusCancelled EventStatus = "cancelled"
)

func (e *EventStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = EventStatus(s)
	case string:
		*e = EventStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for EventStatus: %T", src)
	}
	return nil
}

func (e EventStatus) Value() (driver.Value, error) {
	return string(e), nil
}

type CheckInPhase string

const (
	CheckInPhaseNone   CheckInPhase = "none"
	CheckInPhaseLive   CheckInPhase = "live"
	CheckInPhaseGrace  CheckInPhase = "grace"
	CheckInPhaseLocked CheckInPhase = "locked"
)

type CheckInMethod string

const (
	CheckInMethodSelf          CheckInMethod = "self"
	CheckInMethodManual        CheckInMethod = "manual"
	CheckInMethodAuto          CheckInMethod = "auto"
	CheckInMethodLate          CheckInMethod = "late"
	CheckInMethodStaffOverride CheckInMethod = "staff_override"
)

func (e *CheckInMethod) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = CheckInMethod(s)
	case string:
		*e = CheckInMethod(s)
	default:
		return fmt.Errorf("unsupported scan type for CheckInMethod: %T", src)
	}
	return nil
}

func (e CheckInMethod) Value() (driver.Value, error) {
	return string(e), nil
}

type RoomType string

const (
	RoomTypeClassroom   RoomType = "classroom"
	RoomTypeLaboratory  RoomType = "laboratory"
	RoomTypeLibrary     RoomType = "library"
	RoomTypeOffice      RoomType = "office"
	RoomTypeCafeteria   RoomType = "cafeteria"
	RoomTypeGym         RoomType = "gym"
	RoomTypeAuditorium  RoomType = "auditorium"
	RoomTypeStudyRoom   RoomType = "study_room"
	RoomTypeCommonArea  RoomType = "common_area"
	RoomTypeMedical     RoomType = "medical"
	RoomTypeOther       RoomType = "other"
	RoomTypeLectureHall RoomType = "lecture_hall"
	RoomTypeComputerLab RoomType = "computer_lab"
)

func (e *RoomType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = RoomType(s)
	case string:
		*e = RoomType(s)
	default:
		return fmt.Errorf("unsupported scan type for RoomType: %T", src)
	}
	return nil
}

func (e RoomType) Value() (driver.Value, error) {
	return string(e), nil
}

type ReminderType string

const (
	ReminderTypeDayBefore  ReminderType = "day_before"
	ReminderTypeHourBefore ReminderType = "hour_before"
	ReminderType15Min      ReminderType = "15_min"
)

func (e *ReminderType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = ReminderType(s)
	case string:
		*e = ReminderType(s)
	default:
		return fmt.Errorf("unsupported scan type for ReminderType: %T", src)
	}
	return nil
}

func (e ReminderType) Value() (driver.Value, error) {
	return string(e), nil
}

type ActivityAction string

const (
	ActionEventCreated            ActivityAction = "event.created"
	ActionEventUpdated            ActivityAction = "event.updated"
	ActionEventPublished          ActivityAction = "event.published"
	ActionEventCancelled          ActivityAction = "event.cancelled"
	ActionEventDeleted            ActivityAction = "event.deleted"
	ActionParticipantRegistered   ActivityAction = "participant.registered"
	ActionParticipantUnregistered ActivityAction = "participant.unregistered"
	ActionOrgCreated              ActivityAction = "org.created"
	ActionOrgUpdated              ActivityAction = "org.updated"
	ActionOrgDeleted              ActivityAction = "org.deleted"
	ActionMemberInvited           ActivityAction = "member.invited"
	ActionMemberRemoved           ActivityAction = "member.removed"
	ActionMemberRoleChanged       ActivityAction = "member.role_changed"
	ActionRoomBookingRequested    ActivityAction = "room.booking_requested"
	ActionRoomBookingApproved     ActivityAction = "room.booking_approved"
	ActionRoomBookingRejected     ActivityAction = "room.booking_rejected"
)

func (e *ActivityAction) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = ActivityAction(s)
	case string:
		*e = ActivityAction(s)
	default:
		return fmt.Errorf("unsupported scan type for ActivityAction: %T", src)
	}
	return nil
}

func (e ActivityAction) Value() (driver.Value, error) {
	return string(e), nil
}
