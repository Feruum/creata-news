package activity

import "github.com/studyverse/ems-backend/internal/models"

// Re-export action constants for ergonomic usage at call sites.
// Services import `activity` package, not `models`.
type ActivityAction = models.ActivityAction

const (
	ActionEventCreated            = models.ActionEventCreated
	ActionEventUpdated            = models.ActionEventUpdated
	ActionEventPublished          = models.ActionEventPublished
	ActionEventCancelled          = models.ActionEventCancelled
	ActionEventDeleted            = models.ActionEventDeleted
	ActionParticipantRegistered   = models.ActionParticipantRegistered
	ActionParticipantUnregistered = models.ActionParticipantUnregistered
	ActionOrgCreated              = models.ActionOrgCreated
	ActionOrgUpdated              = models.ActionOrgUpdated
	ActionOrgDeleted              = models.ActionOrgDeleted
	ActionMemberInvited           = models.ActionMemberInvited
	ActionMemberRemoved           = models.ActionMemberRemoved
	ActionMemberRoleChanged       = models.ActionMemberRoleChanged
	ActionRoomBookingRequested    = models.ActionRoomBookingRequested
	ActionRoomBookingApproved     = models.ActionRoomBookingApproved
	ActionRoomBookingRejected     = models.ActionRoomBookingRejected
)

// LogEntry is the input to ActivityLogger.Log().
type LogEntry struct {
	Action         ActivityAction
	ActorID        *int32
	ActorSnapshot  map[string]any
	EntityType     string
	EntityID       int32
	EntitySnapshot map[string]any
	OrganizationID *int32
	OrgSnapshot    map[string]any
	Metadata       map[string]any
}
