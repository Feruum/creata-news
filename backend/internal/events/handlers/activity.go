package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/activity"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

// insertActivityLog is a small helper that persists one activity row.
func insertActivityLog(deps HandlerDeps, msg *message.Message, entry *models.ActivityLog) error {
	entry.SourceEventID = &msg.UUID
	return consumeOnce(msg.Context(), deps.DB, "activity", msg.UUID, func(tx bun.Tx) error {
		if entry.ActorID != nil && *entry.ActorID <= 0 {
			entry.ActorID = nil
		}
		if err := activity.FillActorSnapshots(msg.Context(), tx, []*models.ActivityLog{entry}); err != nil {
			return err
		}
		if _, err := tx.NewInsert().Model(entry).Exec(msg.Context()); err != nil {
			return fmt.Errorf("insert activity log: %w", err)
		}
		return nil
	})
}

func handleOrganizationCreatedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.OrganizationCreatedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal organization.created: %w", err)
		}
		snap := map[string]any{"name": p.OrganizationTitle}
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionOrgCreated,
			ActorID:              &p.ActorID,
			EntityType:           "organization",
			EntityID:             p.OrganizationID,
			EntitySnapshot:       snap,
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{"name": p.OrganizationTitle},
			Metadata:             activity.OrgCreatedMeta(""),
		})
	}
}

func handleOrganizationUpdatedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.OrganizationUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal organization.updated: %w", err)
		}
		snap := map[string]any{"name": p.OrganizationTitle}
		changes := map[string]any{}
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionOrgUpdated,
			ActorID:              &p.ActorID,
			EntityType:           "organization",
			EntityID:             p.OrganizationID,
			EntitySnapshot:       snap,
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{"name": p.OrganizationTitle},
			Metadata:             activity.OrgUpdatedMeta(changes),
		})
	}
}

func handleOrganizationDeletedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.OrganizationDeletedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal organization.deleted: %w", err)
		}
		snap := map[string]any{"name": p.OrganizationTitle}
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionOrgDeleted,
			ActorID:              &p.ActorID,
			EntityType:           "organization",
			EntityID:             p.OrganizationID,
			EntitySnapshot:       snap,
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{"name": p.OrganizationTitle},
			Metadata:             activity.OrgDeletedMeta(),
		})
	}
}

func handleEventCreatedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.EventCreatedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal event.created: %w", err)
		}
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionEventCreated,
			ActorID:              &p.ActorID,
			EntityType:           "event",
			EntityID:             p.EventID,
			EntitySnapshot:       map[string]any{"title": p.Title},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.EventCreatedMeta("", p.Status),
		})
	}
}

func handleEventUpdatedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.EventUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal event.updated: %w", err)
		}

		action := activity.ActionEventUpdated
		var meta map[string]any
		if p.Status == "cancelled" {
			action = activity.ActionEventCancelled
			meta = activity.EventCancelledMeta("")
		} else {
			changes := map[string]any{}
			meta = activity.EventUpdatedMeta(changes)
		}

		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               action,
			ActorID:              &p.ActorID,
			EntityType:           "event",
			EntityID:             p.EventID,
			EntitySnapshot:       map[string]any{"title": p.Title},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             meta,
		})
	}
}

func handleEventDeletedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.EventDeletedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal event.deleted: %w", err)
		}
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionEventDeleted,
			ActorID:              &p.ActorID,
			EntityType:           "event",
			EntityID:             p.EventID,
			EntitySnapshot:       map[string]any{"title": p.Title},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.EventDeletedMeta(),
		})
	}
}

func handleEventPublishedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.EventPublishedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal event.published: %w", err)
		}
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionEventPublished,
			ActorID:              &p.ActorID,
			EntityType:           "event",
			EntityID:             p.EventID,
			EntitySnapshot:       map[string]any{"title": p.Title},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.EventPublishedMeta(),
		})
	}
}

func handleBookingApprovedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.BookingApprovedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal booking.approved: %w", err)
		}
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionRoomBookingApproved,
			ActorID:              &p.ActorID,
			EntityType:           "organization",
			EntityID:             p.OrganizationID,
			EntitySnapshot:       map[string]any{},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.RoomBookingApprovedMeta(p.EventTitle),
		})
	}
}

func handleBookingRejectedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.BookingRejectedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal booking.rejected: %w", err)
		}
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionRoomBookingRejected,
			ActorID:              &p.ActorID,
			EntityType:           "organization",
			EntityID:             p.OrganizationID,
			EntitySnapshot:       map[string]any{},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.RoomBookingRejectedMeta(p.EventTitle, ""),
		})
	}
}

func handleMemberAddedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.MemberAddedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal member.added: %w", err)
		}
		actorID := resolveActorID(p.PerformedBy, p.ActorID)
		memberName := fmt.Sprintf("user:%d", p.UserID)
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionMemberInvited,
			ActorID:              actorID,
			EntityType:           "organization",
			EntityID:             p.OrganizationID,
			EntitySnapshot:       map[string]any{},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.MemberInvitedMeta(memberName, p.RoleName),
		})
	}
}

func handleMemberUpdatedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.MemberUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal member.updated: %w", err)
		}
		actorID := resolveActorID(p.PerformedBy, p.ActorID)
		memberName := fmt.Sprintf("user:%d", p.UserID)
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionMemberRoleChanged,
			ActorID:              actorID,
			EntityType:           "organization",
			EntityID:             p.OrganizationID,
			EntitySnapshot:       map[string]any{},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.MemberRoleChangedMeta(memberName, "", p.RoleName),
		})
	}
}

func handleMemberRemovedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.MemberRemovedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal member.removed: %w", err)
		}
		actorID := resolveActorID(p.PerformedBy, p.ActorID)
		memberName := fmt.Sprintf("user:%d", p.UserID)
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionMemberRemoved,
			ActorID:              actorID,
			EntityType:           "organization",
			EntityID:             p.OrganizationID,
			EntitySnapshot:       map[string]any{},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.MemberRemovedMeta(memberName),
		})
	}
}

func handleRegistrationCreatedActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.RegistrationCreatedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal registration.created: %w", err)
		}
		participantName := fmt.Sprintf("user:%d", p.ParticipantUserID)
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionParticipantRegistered,
			ActorID:              &p.ActorID,
			EntityType:           "event",
			EntityID:             p.EventID,
			EntitySnapshot:       map[string]any{"title": p.EventTitle},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.ParticipantRegisteredMeta(participantName),
		})
	}
}

func handleRegistrationCancelledActivity(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		var p events.RegistrationCancelledPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("unmarshal registration.cancelled: %w", err)
		}
		participantName := fmt.Sprintf("user:%d", p.ParticipantUserID)
		return insertActivityLog(deps, msg, &models.ActivityLog{
			Action:               activity.ActionParticipantUnregistered,
			ActorID:              &p.ActorID,
			EntityType:           "event",
			EntityID:             p.EventID,
			EntitySnapshot:       map[string]any{"title": p.EventTitle},
			OrganizationID:       &p.OrganizationID,
			OrganizationSnapshot: map[string]any{},
			Metadata:             activity.ParticipantUnregisteredMeta(participantName),
		})
	}
}

// resolveActorID returns PerformedBy if set, otherwise ActorID.
// Both fields are optional pointers in member payloads.
func resolveActorID(performedBy, actorID *int32) *int32 {
	if performedBy != nil {
		return performedBy
	}
	return actorID
}
