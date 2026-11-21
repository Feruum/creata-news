package events

const (
	TopicUserCreated           = "user.created"
	TopicUserUpdated           = "user.updated"
	TopicUserDeleted           = "user.deleted"
	TopicOrganizationCreated   = "organization.created"
	TopicOrganizationUpdated   = "organization.updated"
	TopicOrganizationDeleted   = "organization.deleted"
	TopicEventCreated          = "event.created"
	TopicEventUpdated          = "event.updated"
	TopicEventDeleted          = "event.deleted"
	TopicEventPublished        = "event.published"
	TopicEventArchived         = "event.archived"
	TopicCollaborationInvited  = "collaboration.invited"
	TopicCollaborationAccepted = "collaboration.accepted"
	TopicCollaborationDeclined = "collaboration.declined"
	TopicBookingApproved       = "booking.approved"
	TopicBookingRejected       = "booking.rejected"
	TopicMemberAdded           = "member.added"
	TopicMemberUpdated         = "member.updated"
	TopicMemberRemoved         = "member.removed"
	TopicRegistrationCreated   = "registration.created"
	TopicRegistrationCancelled = "registration.cancelled"
	TopicPoison                = "poison"
)

var AllTopics = []string{
	TopicUserCreated,
	TopicUserUpdated,
	TopicUserDeleted,
	TopicOrganizationCreated,
	TopicOrganizationUpdated,
	TopicOrganizationDeleted,
	TopicEventCreated,
	TopicEventUpdated,
	TopicEventDeleted,
	TopicEventPublished,
	TopicEventArchived,
	TopicCollaborationInvited,
	TopicCollaborationAccepted,
	TopicCollaborationDeclined,
	TopicBookingApproved,
	TopicBookingRejected,
	TopicMemberAdded,
	TopicMemberUpdated,
	TopicMemberRemoved,
	TopicRegistrationCreated,
	TopicRegistrationCancelled,
}
