package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/search"
)

func handleUserCreated(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.OnUserCreated != nil {
			deps.OnUserCreated()
		}
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.UserCreatedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal user.created: %w", err)
		}

		return deps.SearchClient.IndexUser(msg.Context(), &search.UserDocument{
			ID:       payload.UserID,
			Username: payload.Username,
			Email:    payload.Email,
		})
	}
}

func handleUserUpdated(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.UserUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal user.updated: %w", err)
		}

		return deps.SearchClient.IndexUser(msg.Context(), &search.UserDocument{
			ID:       payload.UserID,
			Username: payload.Username,
			Email:    payload.Email,
		})
	}
}

func handleUserDeleted(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.UserDeletedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal user.deleted: %w", err)
		}

		return deps.SearchClient.DeleteDocument(msg.Context(), search.IndexUsers, payload.UserID)
	}
}

func handleOrganizationCreated(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.OrganizationCreatedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal organization.created: %w", err)
		}

		return deps.SearchClient.IndexOrganization(msg.Context(), &search.OrganizationDocument{
			ID:                 payload.OrganizationID,
			Title:              payload.OrganizationTitle,
			OrganizationTypeID: payload.TypeID,
		})
	}
}

func handleOrganizationUpdated(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.OrganizationUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal organization.updated: %w", err)
		}

		return deps.SearchClient.IndexOrganization(msg.Context(), &search.OrganizationDocument{
			ID:                 payload.OrganizationID,
			Title:              payload.OrganizationTitle,
			OrganizationTypeID: payload.TypeID,
		})
	}
}

func handleOrganizationDeleted(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.OrganizationDeletedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal organization.deleted: %w", err)
		}

		return deps.SearchClient.DeleteDocument(msg.Context(), search.IndexOrganizations, payload.OrganizationID)
	}
}

func handleEventCreated(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.EventCreatedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal event.created: %w", err)
		}
		if deps.DB != nil {
			return refreshEventSearch(msg.Context(), deps, payload.EventID)
		}
		if payload.Status != "published" {
			return deps.SearchClient.DeleteDocument(msg.Context(), search.IndexEvents, payload.EventID)
		}

		return deps.SearchClient.IndexEvent(msg.Context(), &search.EventDocument{
			ID:                payload.EventID,
			Title:             payload.Title,
			Location:          payload.Location,
			OrganizationID:    payload.OrganizationID,
			OrganizationTitle: payload.OrganizationTitle,
			StartTime:         payload.StartTime,
			EndTime:           payload.EndTime,
		})
	}
}

func handleEventUpdated(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.EventUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal event.updated: %w", err)
		}
		if deps.DB != nil {
			return refreshEventSearch(msg.Context(), deps, payload.EventID)
		}

		if payload.Status != "published" {
			return deps.SearchClient.DeleteDocument(msg.Context(), search.IndexEvents, payload.EventID)
		}

		return deps.SearchClient.IndexEvent(msg.Context(), &search.EventDocument{
			ID:                payload.EventID,
			Title:             payload.Title,
			Location:          payload.Location,
			OrganizationID:    payload.OrganizationID,
			OrganizationTitle: payload.OrganizationTitle,
			StartTime:         payload.StartTime,
			EndTime:           payload.EndTime,
		})
	}
}

func handleEventPublishedSearch(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}
		var payload events.EventPublishedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal event.published: %w", err)
		}
		return refreshEventSearch(msg.Context(), deps, payload.EventID)
	}
}

func handleEventArchivedSearch(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}
		var payload events.EventArchivedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal event.archived: %w", err)
		}
		return refreshEventSearch(msg.Context(), deps, payload.EventID)
	}
}

func handleEventDeleted(deps HandlerDeps) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		if deps.SearchClient == nil {
			return nil
		}

		var payload events.EventDeletedPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal event.deleted: %w", err)
		}
		if deps.DB != nil {
			return refreshEventSearch(msg.Context(), deps, payload.EventID)
		}

		return deps.SearchClient.DeleteDocument(msg.Context(), search.IndexEvents, payload.EventID)
	}
}
