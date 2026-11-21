package events

type EventArchivedPayload struct {
	EventID int32 `json:"event_id"`
}

type UserCreatedPayload struct {
	UserID   int32  `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserUpdatedPayload struct {
	UserID   int32  `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserDeletedPayload struct {
	UserID int32 `json:"user_id"`
}

type OrganizationCreatedPayload struct {
	OrganizationID    int32  `json:"organization_id"`
	OrganizationTitle string `json:"organization_title"`
	TypeID            int32  `json:"type_id"`
	ActorID           int32  `json:"actor_id"`
}

type OrganizationUpdatedPayload struct {
	OrganizationID    int32  `json:"organization_id"`
	OrganizationTitle string `json:"organization_title"`
	TypeID            int32  `json:"type_id"`
	IsArchived        bool   `json:"is_archived"`
	ActorID           int32  `json:"actor_id"`
}

type OrganizationDeletedPayload struct {
	OrganizationID    int32  `json:"organization_id"`
	OrganizationTitle string `json:"organization_title"`
	ActorID           int32  `json:"actor_id"`
}

type EventCreatedPayload struct {
	EventID           int32  `json:"event_id"`
	Title             string `json:"title"`
	OrganizationID    int32  `json:"organization_id"`
	OrganizationTitle string `json:"organization_title"`
	CategoryID        int32  `json:"category_id"`
	CategoryName      string `json:"category_name"`
	Status            string `json:"status"`
	StartTime         string `json:"start_time"`
	EndTime           string `json:"end_time"`
	Location          string `json:"location"`
	EventSlug         string `json:"event_slug"`
	ActorID           int32  `json:"actor_id"`
}

type EventUpdatedPayload struct {
	EventID           int32  `json:"event_id"`
	Title             string `json:"title"`
	OrganizationID    int32  `json:"organization_id"`
	OrganizationTitle string `json:"organization_title"`
	CategoryID        int32  `json:"category_id"`
	CategoryName      string `json:"category_name"`
	Status            string `json:"status"`
	StartTime         string `json:"start_time"`
	EndTime           string `json:"end_time"`
	Location          string `json:"location"`
	EventSlug         string `json:"event_slug"`
	ActorID           int32  `json:"actor_id"`
}

type EventDeletedPayload struct {
	EventID        int32  `json:"event_id"`
	Title          string `json:"title"`
	OrganizationID int32  `json:"organization_id"`
	ActorID        int32  `json:"actor_id"`
}

type EventPublishedPayload struct {
	EventID          int32  `json:"event_id"`
	Title            string `json:"title"`
	OrganizationID   int32  `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	StartTime        string `json:"start_time"`
	Location         string `json:"location"`
	EventSlug        string `json:"event_slug"`
	MapAppURL        string `json:"map_app_url"`
	ActorID          int32  `json:"actor_id"`
}

type CollaborationInvitedPayload struct {
	EventID            int32  `json:"event_id"`
	EventTitle         string `json:"event_title"`
	OrganizationID     int32  `json:"organization_id"`
	HostOrganizationID int32  `json:"host_organization_id"`
	InviteToken        string `json:"invite_token"`
	MapAppURL          string `json:"map_app_url"`
	ActorID            int32  `json:"actor_id"`
}

type CollaborationAcceptedPayload struct {
	EventID            int32  `json:"event_id"`
	EventTitle         string `json:"event_title"`
	OrganizationID     int32  `json:"organization_id"`
	HostOrganizationID int32  `json:"host_organization_id"`
	ActorID            int32  `json:"actor_id"`
}

type CollaborationDeclinedPayload struct {
	EventID            int32  `json:"event_id"`
	EventTitle         string `json:"event_title"`
	OrganizationID     int32  `json:"organization_id"`
	HostOrganizationID int32  `json:"host_organization_id"`
	ActorID            int32  `json:"actor_id"`
}

type BookingApprovedPayload struct {
	BookingRequestID int32  `json:"booking_request_id"`
	EventID          int32  `json:"event_id"`
	EventTitle       string `json:"event_title"`
	OrganizationID   int32  `json:"organization_id"`
	RoomID           string `json:"room_id"`
	RoomName         string `json:"room_name"`
	ActorID          int32  `json:"actor_id"`
}

type BookingRejectedPayload struct {
	BookingRequestID int32  `json:"booking_request_id"`
	EventID          int32  `json:"event_id"`
	EventTitle       string `json:"event_title"`
	OrganizationID   int32  `json:"organization_id"`
	RoomID           string `json:"room_id"`
	RoomName         string `json:"room_name"`
	ActorID          int32  `json:"actor_id"`
}

type MemberAddedPayload struct {
	UserID            int32  `json:"user_id"`
	OrganizationID    int32  `json:"organization_id"`
	OrganizationTitle string `json:"organization_title"`
	RoleTemplateID    int32  `json:"role_template_id"`
	RoleName          string `json:"role_name"`
	PerformedBy       *int32 `json:"performed_by,omitempty"`
	ActorID           *int32 `json:"actor_id,omitempty"`
}

type MemberUpdatedPayload struct {
	UserID            int32  `json:"user_id"`
	OrganizationID    int32  `json:"organization_id"`
	OrganizationTitle string `json:"organization_title"`
	RoleTemplateID    int32  `json:"role_template_id"`
	RoleName          string `json:"role_name"`
	PerformedBy       *int32 `json:"performed_by,omitempty"`
	ActorID           *int32 `json:"actor_id,omitempty"`
}

type MemberRemovedPayload struct {
	UserID            int32  `json:"user_id"`
	OrganizationID    int32  `json:"organization_id"`
	OrganizationTitle string `json:"organization_title"`
	RoleTemplateID    int32  `json:"role_template_id"`
	RoleName          string `json:"role_name"`
	PerformedBy       *int32 `json:"performed_by,omitempty"`
	ActorID           *int32 `json:"actor_id,omitempty"`
}

type RegistrationCreatedPayload struct {
	RegistrationID    int32  `json:"registration_id"`
	EventID           int32  `json:"event_id"`
	EventTitle        string `json:"event_title"`
	OrganizationID    int32  `json:"organization_id"`
	ParticipantUserID int32  `json:"participant_user_id"`
	ActorID           int32  `json:"actor_id"`
}

type RegistrationCancelledPayload struct {
	RegistrationID    int32  `json:"registration_id"`
	EventID           int32  `json:"event_id"`
	EventTitle        string `json:"event_title"`
	OrganizationID    int32  `json:"organization_id"`
	ParticipantUserID int32  `json:"participant_user_id"`
	ActorID           int32  `json:"actor_id"`
}
