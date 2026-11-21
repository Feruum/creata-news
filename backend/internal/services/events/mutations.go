package events

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/uptrace/bun"
)

func runEventMutation[T any](ctx context.Context, s *Service, fn func(*Service) (*connect.Response[T], error)) (*connect.Response[T], error) {
	if auth.GetUserID(ctx) == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	var result *connect.Response[T]
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		scoped := s.InTransaction(tx)
		var err error
		result, err = fn(scoped)
		if err != nil {
			return err
		}
		if scoped.publisher != nil && scoped.publisher.Err() != nil {
			return connect.NewError(connect.CodeInternal, scoped.publisher.Err())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// InTransaction lets a private workflow commit its receipt with the normal event mutation and outbox.
func (s *Service) InTransaction(tx bun.Tx) *Service {
	scoped := *s
	scoped.db = tx
	scoped.authz = s.authz.InTransaction(tx)
	scoped.publisher = s.publisher.InTransaction(tx)
	return &scoped
}

func (s *Service) CreateEvent(ctx context.Context, req *connect.Request[eventsv1.CreateEventRequest]) (*connect.Response[eventsv1.CreateEventResponse], error) {
	return runEventMutation(ctx, s, func(scoped *Service) (*connect.Response[eventsv1.CreateEventResponse], error) {
		return scoped.createEvent(ctx, req)
	})
}
func (s *Service) UpdateEvent(ctx context.Context, req *connect.Request[eventsv1.UpdateEventRequest]) (*connect.Response[eventsv1.UpdateEventResponse], error) {
	return runEventMutation(ctx, s, func(scoped *Service) (*connect.Response[eventsv1.UpdateEventResponse], error) {
		return scoped.updateEvent(ctx, req)
	})
}
func (s *Service) PublishEvent(ctx context.Context, req *connect.Request[eventsv1.PublishEventRequest]) (*connect.Response[eventsv1.PublishEventResponse], error) {
	return runEventMutation(ctx, s, func(scoped *Service) (*connect.Response[eventsv1.PublishEventResponse], error) {
		return scoped.publishEvent(ctx, req)
	})
}
func (s *Service) DeleteEvent(ctx context.Context, req *connect.Request[eventsv1.DeleteEventRequest]) (*connect.Response[eventsv1.DeleteEventResponse], error) {
	return runEventMutation(ctx, s, func(scoped *Service) (*connect.Response[eventsv1.DeleteEventResponse], error) {
		return scoped.deleteEvent(ctx, req)
	})
}
