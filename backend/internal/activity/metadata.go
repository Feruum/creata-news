package activity

// EventCreatedMeta returns metadata for event.created.
func EventCreatedMeta(format string, status string) map[string]any {
	return map[string]any{"format": format, "status": status}
}

// EventUpdatedMeta returns metadata for event.updated.
// changes is a map of field name → {old, new} pairs.
func EventUpdatedMeta(changes map[string]any) map[string]any {
	return map[string]any{"changes": changes}
}

// EventPublishedMeta returns metadata for event.published.
func EventPublishedMeta() map[string]any {
	return map[string]any{}
}

// EventCancelledMeta returns metadata for event.cancelled.
func EventCancelledMeta(reason string) map[string]any {
	m := map[string]any{}
	if reason != "" {
		m["reason"] = reason
	}
	return m
}

// EventDeletedMeta returns metadata for event.deleted.
func EventDeletedMeta() map[string]any {
	return map[string]any{}
}

// ParticipantRegisteredMeta returns metadata for participant.registered.
func ParticipantRegisteredMeta(participantName string) map[string]any {
	return map[string]any{"participant_name": participantName}
}

// ParticipantUnregisteredMeta returns metadata for participant.unregistered.
func ParticipantUnregisteredMeta(participantName string) map[string]any {
	return map[string]any{"participant_name": participantName}
}

// OrgCreatedMeta returns metadata for org.created.
func OrgCreatedMeta(orgType string) map[string]any {
	return map[string]any{"org_type": orgType}
}

// OrgUpdatedMeta returns metadata for org.updated.
// changes is a map of field name → {old, new} pairs.
func OrgUpdatedMeta(changes map[string]any) map[string]any {
	return map[string]any{"changes": changes}
}

// OrgDeletedMeta returns metadata for org.deleted.
func OrgDeletedMeta() map[string]any {
	return map[string]any{}
}

// MemberInvitedMeta returns metadata for member.invited.
func MemberInvitedMeta(memberName string, role string) map[string]any {
	return map[string]any{"member_name": memberName, "role": role}
}

// MemberRemovedMeta returns metadata for member.removed.
func MemberRemovedMeta(memberName string) map[string]any {
	return map[string]any{"member_name": memberName}
}

// MemberRoleChangedMeta returns metadata for member.role_changed.
func MemberRoleChangedMeta(memberName string, oldRole string, newRole string) map[string]any {
	return map[string]any{"member_name": memberName, "old_role": oldRole, "new_role": newRole}
}

// RoomBookingRequestedMeta returns metadata for room.booking_requested.
func RoomBookingRequestedMeta(eventTitle string, date string) map[string]any {
	return map[string]any{"event_title": eventTitle, "date": date}
}

// RoomBookingApprovedMeta returns metadata for room.booking_approved.
func RoomBookingApprovedMeta(eventTitle string) map[string]any {
	return map[string]any{"event_title": eventTitle}
}

// RoomBookingRejectedMeta returns metadata for room.booking_rejected.
func RoomBookingRejectedMeta(eventTitle string, reason string) map[string]any {
	m := map[string]any{"event_title": eventTitle}
	if reason != "" {
		m["reason"] = reason
	}
	return m
}
