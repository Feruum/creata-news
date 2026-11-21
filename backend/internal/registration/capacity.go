package registration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

// Both registration channels serialize seat allocation on the same event row.
// The caller must hold this transaction until the registration has committed.
func LockOpenEvent(ctx context.Context, tx bun.Tx, eventID int32, now func() time.Time) (*models.Event, error) {
	var event models.Event
	err := tx.NewSelect().Model(&event).Where("id = ?", eventID).For("UPDATE").Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("event not found"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("lock registration event: %w", err))
	}
	if event.Status != models.EventStatusPublished || !event.EndTime.After(now()) || event.CompletedAt != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("event registration is closed"))
	}
	return &event, nil
}

func EnsureCapacity(ctx context.Context, tx bun.Tx, event *models.Event) error {
	if event.Capacity == nil {
		return nil
	}
	var count int
	err := tx.NewRaw(`SELECT
		(SELECT count(*) FROM event_registrations WHERE event_id=? AND status='registered') +
		(SELECT count(*) FROM external_registrations WHERE event_id=? AND status='registered')`, event.ID, event.ID).Scan(ctx, &count)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("count allocated seats: %w", err))
	}
	if count >= int(*event.Capacity) {
		return connect.NewError(connect.CodeResourceExhausted, errors.New("event capacity has been reached"))
	}
	return nil
}
