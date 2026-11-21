package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type BookingRequest struct {
	bun.BaseModel `bun:"table:booking_requests,alias:br"`

	ID              int32                `bun:"id,pk,autoincrement" json:"id"`
	RoomID          string               `bun:"room_id,notnull" json:"room_id"`
	RequesterID     int32                `bun:"requester_id,notnull" json:"requester_id"`
	OrganizationID  *int32               `bun:"organization_id" json:"organization_id"`
	Priority        BookingPriority      `bun:"priority,notnull,type:booking_priority" json:"priority"`
	Title           string               `bun:"title,notnull" json:"title"`
	Description     *string              `bun:"description" json:"description"`
	StartTime       time.Time            `bun:"start_time,notnull" json:"start_time"`
	EndTime         time.Time            `bun:"end_time,notnull" json:"end_time"`
	Status          BookingRequestStatus `bun:"status,notnull,type:booking_request_status" json:"status"`
	ReviewedBy      *int32               `bun:"reviewed_by" json:"reviewed_by"`
	ReviewedAt      *time.Time           `bun:"reviewed_at" json:"reviewed_at"`
	RejectionReason *string              `bun:"rejection_reason" json:"rejection_reason"`
	EventID         *int32               `bun:"event_id" json:"event_id"`
	CreatedAt       time.Time            `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time            `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type Category struct {
	bun.BaseModel `bun:"table:categories,alias:c"`

	ID        int32     `bun:"id,pk,autoincrement" json:"id"`
	Name      string    `bun:"name,notnull" json:"name"`
	Color     string    `bun:"color,notnull" json:"color"`
	CreatedAt time.Time `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,default:current_timestamp" json:"updated_at"`
}

type Event struct {
	bun.BaseModel `bun:"table:events,alias:e"`

	ID                    int32           `bun:"id,pk,autoincrement" json:"id"`
	Title                 string          `bun:"title,notnull" json:"title"`
	Description           json.RawMessage `bun:"description,notnull" json:"description"`
	ImageUrl              *string         `bun:"image_url" json:"image_url"`
	UserID                int32           `bun:"user_id,notnull" json:"user_id"`
	OrganizationID        int32           `bun:"organization_id,notnull" json:"organization_id"`
	StartTime             time.Time       `bun:"start_time,notnull" json:"start_time"`
	EndTime               time.Time       `bun:"end_time,notnull" json:"end_time"`
	Format                *Format         `bun:"format,type:format" json:"format"`
	CreatedAt             time.Time       `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt             time.Time       `bun:"updated_at,default:current_timestamp" json:"updated_at"`
	CategoryID            *int32          `bun:"category_id" json:"category_id"`
	Capacity              *int32          `bun:"capacity" json:"capacity"`
	PublishOnMap          bool            `bun:"publish_on_map,notnull" json:"publish_on_map"`
	PublishOnSundayOpen   bool            `bun:"publish_on_sunday_open,notnull" json:"publish_on_sunday_open"`
	StructuredLocation    json.RawMessage `bun:"structured_location,notnull" json:"structured_location"`
	SocGpaBadgeStatus     string          `bun:"soc_gpa_badge_status,notnull,default:'not_requested'" json:"soc_gpa_badge_status"`
	SocGpaBadgeReviewedBy *int32          `bun:"soc_gpa_badge_reviewed_by" json:"soc_gpa_badge_reviewed_by"`
	SocGpaBadgeReviewedAt *time.Time      `bun:"soc_gpa_badge_reviewed_at" json:"soc_gpa_badge_reviewed_at"`
	IrosBadgeStatus       string          `bun:"iros_badge_status,notnull,default:'not_requested'" json:"iros_badge_status"`
	IrosBadgeReviewedBy   *int32          `bun:"iros_badge_reviewed_by" json:"iros_badge_reviewed_by"`
	IrosBadgeReviewedAt   *time.Time      `bun:"iros_badge_reviewed_at" json:"iros_badge_reviewed_at"`

	// Event lifecycle fields
	Status              EventStatus `bun:"status,notnull,type:event_status,default:'published'" json:"status"`
	CheckInGraceMinutes *int32      `bun:"check_in_grace_minutes" json:"check_in_grace_minutes"`
	CheckInSecret       *string     `bun:"check_in_secret" json:"check_in_secret"`
	CheckInToken        *string     `bun:"check_in_token" json:"check_in_token"`
	PublishedAt         *time.Time  `bun:"published_at" json:"published_at"`
	CompletedAt         *time.Time  `bun:"completed_at" json:"completed_at"`
	CancelledAt         *time.Time  `bun:"cancelled_at" json:"cancelled_at"`
	CancelledBy         *int32      `bun:"cancelled_by" json:"cancelled_by"`
	CancelReason        *string     `bun:"cancel_reason" json:"cancel_reason"`
	ArchivedAt          *time.Time  `bun:"archived_at" json:"archived_at"`
}

type EventAttendance struct {
	bun.BaseModel `bun:"table:event_attendance,alias:ea"`

	ID             int32            `bun:"id,pk,autoincrement" json:"id"`
	RegistrationID int32            `bun:"registration_id,notnull" json:"registration_id"`
	Status         AttendanceStatus `bun:"status,notnull,type:attendance_status" json:"status"`
	CheckedInAt    *time.Time       `bun:"checked_in_at" json:"checked_in_at"`
	CheckedInBy    *int32           `bun:"checked_in_by" json:"checked_in_by"`
	Notes          *string          `bun:"notes" json:"notes"`
	Method         *CheckInMethod   `bun:"method,type:check_in_method" json:"method"`
	Reason         *string          `bun:"reason" json:"reason"`
	CreatedAt      time.Time        `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time        `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type EventCollaborationInvite struct {
	bun.BaseModel `bun:"table:event_collaboration_invites,alias:ci"`

	ID                int32                     `bun:"id,pk,autoincrement" json:"id"`
	EventID           int32                     `bun:"event_id,notnull" json:"event_id"`
	OrganizationID    int32                     `bun:"organization_id,notnull" json:"organization_id"`
	InvitedByUserID   int32                     `bun:"invited_by_user_id,notnull" json:"invited_by_user_id"`
	Status            CollaborationInviteStatus `bun:"status,notnull,type:collaboration_invite_status" json:"status"`
	Message           *string                   `bun:"message" json:"message"`
	Token             string                    `bun:"token,notnull" json:"token"`
	ExpiresAt         time.Time                 `bun:"expires_at,notnull" json:"expires_at"`
	RespondedAt       *time.Time                `bun:"responded_at" json:"responded_at"`
	RespondedByUserID *int32                    `bun:"responded_by_user_id" json:"responded_by_user_id"`
	CreatedAt         time.Time                 `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt         time.Time                 `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type EventCollaborator struct {
	bun.BaseModel `bun:"table:event_collaborators,alias:co"`

	ID             int32     `bun:"id,pk,autoincrement" json:"id"`
	EventID        int32     `bun:"event_id,notnull" json:"event_id"`
	OrganizationID int32     `bun:"organization_id,notnull" json:"organization_id"`
	AddedBy        *int32    `bun:"added_by" json:"added_by"`
	AddedAt        time.Time `bun:"added_at,notnull" json:"added_at"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type EventOrganizer struct {
	bun.BaseModel `bun:"table:event_organizers,alias:eo"`

	ID             int32     `bun:"id,pk,autoincrement" json:"id"`
	EventID        int32     `bun:"event_id,notnull" json:"event_id"`
	UserID         int32     `bun:"user_id,notnull" json:"user_id"`
	OrganizationID int32     `bun:"organization_id,notnull" json:"organization_id"`
	Role           *string   `bun:"role" json:"role"`
	AddedBy        *int32    `bun:"added_by" json:"added_by"`
	AddedAt        time.Time `bun:"added_at,notnull" json:"added_at"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type EventRegistration struct {
	bun.BaseModel `bun:"table:event_registrations,alias:er"`

	ID           int32              `bun:"id,pk,autoincrement" json:"id"`
	EventID      int32              `bun:"event_id,notnull" json:"event_id"`
	UserID       int32              `bun:"user_id,notnull" json:"user_id"`
	Status       RegistrationStatus `bun:"status,notnull,type:registration_status" json:"status"`
	RegisteredAt time.Time          `bun:"registered_at,notnull" json:"registered_at"`
	CancelledAt  *time.Time         `bun:"cancelled_at" json:"cancelled_at"`
	CreatedAt    time.Time          `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt    time.Time          `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type ExternalRegistration struct {
	bun.BaseModel `bun:"table:external_registrations,alias:xr"`

	ID                  int32                      `bun:"id,pk,autoincrement" json:"id"`
	EventID             int32                      `bun:"event_id,notnull" json:"event_id"`
	FullName            string                     `bun:"full_name,notnull" json:"full_name"`
	Phone               *string                    `bun:"phone" json:"phone"`
	SundayOpenAccountID *string                    `bun:"sunday_open_account_id" json:"sunday_open_account_id"`
	CheckedInAt         *time.Time                 `bun:"checked_in_at" json:"checked_in_at"`
	Snapshot            json.RawMessage            `bun:"snapshot,type:jsonb,default:'{}'" json:"snapshot"`
	Email               string                     `bun:"email,notnull" json:"email"`
	Status              ExternalRegistrationStatus `bun:"status,notnull,type:external_registration_status" json:"status"`
	RegisteredAt        time.Time                  `bun:"registered_at,notnull" json:"registered_at"`
	CancelledAt         *time.Time                 `bun:"cancelled_at" json:"cancelled_at"`
	CreatedAt           time.Time                  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt           time.Time                  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type EventSlug struct {
	bun.BaseModel `bun:"table:event_slugs,alias:es"`

	ID        int32     `bun:"id,pk,autoincrement" json:"id"`
	EventID   int32     `bun:"event_id,notnull" json:"event_id"`
	Slug      string    `bun:"slug,notnull" json:"slug"`
	IsCurrent bool      `bun:"is_current,notnull" json:"is_current"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

type Organization struct {
	bun.BaseModel `bun:"table:organizations,alias:o"`

	ID                 int32               `bun:"id,pk,autoincrement" json:"id"`
	Title              string              `bun:"title,notnull" json:"title"`
	ImageUrl           *string             `bun:"image_url" json:"image_url"`
	Description        json.RawMessage     `bun:"description" json:"description"`
	OrganizationTypeID int32               `bun:"organization_type_id,notnull" json:"organization_type_id"`
	Instagram          *string             `bun:"instagram" json:"instagram"`
	TelegramChannel    *string             `bun:"telegram_channel" json:"telegram_channel"`
	TelegramChat       *string             `bun:"telegram_chat" json:"telegram_chat"`
	Website            *string             `bun:"website" json:"website"`
	Youtube            *string             `bun:"youtube" json:"youtube"`
	Tiktok             *string             `bun:"tiktok" json:"tiktok"`
	Linkedin           *string             `bun:"linkedin" json:"linkedin"`
	Status             *OrganizationStatus `bun:"status,type:organization_status" json:"status"`
	CreatedAt          time.Time           `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt          time.Time           `bun:"updated_at,default:current_timestamp" json:"updated_at"`
}

type OrganizationMember struct {
	bun.BaseModel `bun:"table:organization_members,alias:om"`

	ID             int32      `bun:"id,pk,autoincrement" json:"id"`
	OrganizationID int32      `bun:"organization_id,notnull" json:"organization_id"`
	UserID         int32      `bun:"user_id,notnull" json:"user_id"`
	RoleTemplateID int32      `bun:"role_template_id,notnull" json:"role_template_id"`
	AssignedBy     *int32     `bun:"assigned_by" json:"assigned_by"`
	AssignedAt     time.Time  `bun:"assigned_at,notnull" json:"assigned_at"`
	ValidUntil     *time.Time `bun:"valid_until" json:"valid_until"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type OrganizationSlug struct {
	bun.BaseModel `bun:"table:organization_slugs,alias:os"`

	ID             int32     `bun:"id,pk,autoincrement" json:"id"`
	OrganizationID int32     `bun:"organization_id,notnull" json:"organization_id"`
	Slug           string    `bun:"slug,notnull" json:"slug"`
	IsCurrent      bool      `bun:"is_current,notnull" json:"is_current"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

type OrganizationType struct {
	bun.BaseModel `bun:"table:organization_types,alias:ot"`

	ID        int32     `bun:"id,pk,autoincrement" json:"id"`
	Title     string    `bun:"title,notnull" json:"title"`
	CreatedAt time.Time `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,default:current_timestamp" json:"updated_at"`
}

type PreRegisteredUser struct {
	bun.BaseModel `bun:"table:pre_registered_users,alias:pr"`

	ID           int32        `bun:"id,pk,autoincrement" json:"id"`
	Email        string       `bun:"email,notnull" json:"email"`
	PlatformRole PlatformRole `bun:"platform_role,notnull,type:platform_role" json:"platform_role"`
	CreatedBy    *int32       `bun:"created_by" json:"created_by"`
	UsedAt       *time.Time   `bun:"used_at" json:"used_at"`
	UsedByUserID *int32       `bun:"used_by_user_id" json:"used_by_user_id"`
	CreatedAt    time.Time    `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt    time.Time    `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type Role struct {
	bun.BaseModel `bun:"table:roles,alias:rl"`

	ID        int32     `bun:"id,pk,autoincrement" json:"id"`
	Name      string    `bun:"name,notnull" json:"name"`
	CreatedAt time.Time `bun:"created_at,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,default:current_timestamp" json:"updated_at"`
}

type RoleTemplate struct {
	bun.BaseModel `bun:"table:role_templates,alias:rt"`

	ID                 int32     `bun:"id,pk,autoincrement" json:"id"`
	OrganizationTypeID int32     `bun:"organization_type_id,notnull" json:"organization_type_id"`
	OrganizationID     *int32    `bun:"organization_id" json:"organization_id"`
	Key                string    `bun:"key,notnull" json:"key"`
	DisplayName        string    `bun:"display_name,notnull" json:"display_name"`
	Description        *string   `bun:"description" json:"description"`
	ParentTemplateID   *int32    `bun:"parent_template_id" json:"parent_template_id"`
	SortOrder          int32     `bun:"sort_order,notnull" json:"sort_order"`
	MaxHolders         *int32    `bun:"max_holders" json:"max_holders"`
	IsRequired         bool      `bun:"is_required,notnull" json:"is_required"`
	IsActive           bool      `bun:"is_active,notnull" json:"is_active"`
	CreatedAt          time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt          time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type RoleTemplateCapability struct {
	bun.BaseModel `bun:"table:role_template_capabilities,alias:rc"`

	ID             int32      `bun:"id,pk,autoincrement" json:"id"`
	RoleTemplateID int32      `bun:"role_template_id,notnull" json:"role_template_id"`
	Capability     Capability `bun:"capability,notnull,type:capability" json:"capability"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

type Room struct {
	bun.BaseModel `bun:"table:rooms,alias:r"`

	ID              string           `bun:"id,pk" json:"id"`
	Name            string           `bun:"name,notnull" json:"name"`
	RoomNumber      string           `bun:"room_number,notnull" json:"room_number"`
	Block           string           `bun:"block,notnull" json:"block"`
	Floor           int32            `bun:"floor,notnull" json:"floor"`
	Type            RoomType         `bun:"type,notnull,type:room_type" json:"type"`
	Capacity        *int32           `bun:"capacity" json:"capacity"`
	Description     *string          `bun:"description" json:"description"`
	Features        *json.RawMessage `bun:"features" json:"features"`
	PrimaryImageUrl *string          `bun:"primary_image_url" json:"primary_image_url"`
	IsAccessible    bool             `bun:"is_accessible,notnull" json:"is_accessible"`
	IsActive        bool             `bun:"is_active,notnull" json:"is_active"`
	IsBookable      bool             `bun:"is_bookable,notnull" json:"is_bookable"`
	CreatedAt       time.Time        `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time        `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type RoomBooking struct {
	bun.BaseModel `bun:"table:room_bookings,alias:rb"`

	ID               int32     `bun:"id,pk,autoincrement" json:"id"`
	RoomID           string    `bun:"room_id,notnull" json:"room_id"`
	BookingRequestID *int32    `bun:"booking_request_id" json:"booking_request_id"`
	EventID          *int32    `bun:"event_id" json:"event_id"`
	CreatedBy        int32     `bun:"created_by,notnull" json:"created_by"`
	Title            string    `bun:"title,notnull" json:"title"`
	StartTime        time.Time `bun:"start_time,notnull" json:"start_time"`
	EndTime          time.Time `bun:"end_time,notnull" json:"end_time"`
	CreatedAt        time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt        time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID               int32      `bun:"id,pk,autoincrement" json:"id"`
	KratosID         *string    `bun:"kratos_id" json:"kratos_id"`
	Username         string     `bun:"username,notnull" json:"username"`
	Email            string     `bun:"email,notnull" json:"email"`
	Password         string     `bun:"password,notnull" json:"password"`
	CreatedAt        time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt        time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	FirstName        *string    `bun:"first_name" json:"first_name"`
	LastName         *string    `bun:"last_name" json:"last_name"`
	AvatarURL        *string    `bun:"avatar_url" json:"avatar_url"`
	TelegramID       *int64     `bun:"telegram_id" json:"telegram_id"`
	TelegramUsername *string    `bun:"telegram_username" json:"telegram_username"`
	TelegramLinkedAt *time.Time `bun:"telegram_linked_at" json:"telegram_linked_at"`
}

type UserRole struct {
	bun.BaseModel `bun:"table:user_roles,alias:ur"`

	ID             int32 `bun:"id,pk,autoincrement" json:"id"`
	UserID         int32 `bun:"user_id,notnull" json:"user_id"`
	OrganizationID int32 `bun:"organization_id,notnull" json:"organization_id"`
	RoleID         int32 `bun:"role_id,notnull" json:"role_id"`
}

type EventReminder struct {
	bun.BaseModel `bun:"table:event_reminders,alias:evr"`

	ID                int32        `bun:"id,pk,autoincrement" json:"id"`
	EventID           int32        `bun:"event_id,notnull" json:"event_id"`
	UserID            int32        `bun:"user_id,notnull" json:"user_id"`
	ReminderType      ReminderType `bun:"reminder_type,notnull,type:reminder_type" json:"reminder_type"`
	SentAt            time.Time    `bun:"sent_at,notnull,default:current_timestamp" json:"sent_at"`
	TelegramMessageID *int64       `bun:"telegram_message_id" json:"telegram_message_id"`
	CreatedAt         time.Time    `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt         time.Time    `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type ActivityLog struct {
	SourceEventID        *string `bun:"source_event_id"`
	bun.BaseModel        `bun:"table:activity_log,alias:al"`
	ID                   int64          `bun:"id,pk,autoincrement"`
	Action               ActivityAction `bun:"action,notnull,type:activity_action"`
	ActorID              *int32         `bun:"actor_id"`
	ActorSnapshot        map[string]any `bun:"actor_snapshot,notnull,type:jsonb,default:'{}'"`
	EntityType           string         `bun:"entity_type,notnull"`
	EntityID             int32          `bun:"entity_id,notnull"`
	EntitySnapshot       map[string]any `bun:"entity_snapshot,notnull,type:jsonb,default:'{}'"`
	OrganizationID       *int32         `bun:"organization_id"`
	OrganizationSnapshot map[string]any `bun:"organization_snapshot,notnull,type:jsonb,default:'{}'"`
	Metadata             map[string]any `bun:"metadata,notnull,type:jsonb,default:'{}'"`
	CreatedAt            time.Time      `bun:"created_at,notnull,default:current_timestamp"`
}

type MembershipHistoryAction string

const (
	MembershipHistoryActionAssigned    MembershipHistoryAction = "assigned"
	MembershipHistoryActionRoleChanged MembershipHistoryAction = "role_changed"
	MembershipHistoryActionRemoved     MembershipHistoryAction = "removed"
	MembershipHistoryActionExpired     MembershipHistoryAction = "expired"
	MembershipHistoryActionOrgDeleted  MembershipHistoryAction = "org_deleted"
	MembershipHistoryActionOrgArchived MembershipHistoryAction = "org_archived"
)

type MembershipHistory struct {
	SourceEventID          *string `bun:"source_event_id"`
	bun.BaseModel          `bun:"table:membership_history,alias:mh"`
	ID                     int64                   `bun:"id,pk,autoincrement"`
	UserID                 int32                   `bun:"user_id,notnull"`
	OrganizationID         *int32                  `bun:"organization_id"`
	Action                 MembershipHistoryAction `bun:"action,notnull,type:membership_history_action"`
	RoleTemplateID         *int32                  `bun:"role_template_id"`
	PreviousRoleTemplateID *int32                  `bun:"previous_role_template_id"`
	PerformedBy            *int32                  `bun:"performed_by"`
	Reason                 *string                 `bun:"reason"`
	RoleName               *string                 `bun:"role_name"`
	PreviousRoleName       *string                 `bun:"previous_role_name"`
	OrganizationTitle      string                  `bun:"organization_title,notnull"`
	EffectiveAt            time.Time               `bun:"effective_at,notnull,default:current_timestamp"`
	CreatedAt              time.Time               `bun:"created_at,notnull,default:current_timestamp"`
}
