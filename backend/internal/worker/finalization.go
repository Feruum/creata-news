package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/uptrace/bun"

	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

type FinalizationWorker struct {
	db       *bun.DB
	interval time.Duration
}

func NewFinalizationWorker(db *bun.DB) *FinalizationWorker {
	return &FinalizationWorker{
		db:       db,
		interval: 15 * time.Minute,
	}
}

func (w *FinalizationWorker) Start(ctx context.Context) {
	slog.Info("Finalization worker started", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Finalization worker stopped")
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

func (w *FinalizationWorker) run(ctx context.Context) {
	now := time.Now()

	finalized, err := w.finalizeEvents(ctx, now)
	if err != nil {
		slog.Error("Finalization worker: failed to finalize events", "error", err)
	} else if finalized > 0 {
		slog.Info("Finalization worker: finalized events", "count", finalized)
	}

	archived, err := w.archiveOldEvents(ctx, now)
	if err != nil {
		slog.Error("Finalization worker: failed to archive old events", "error", err)
	} else if archived > 0 {
		slog.Info("Finalization worker: archived old events", "count", archived)
	}
}

func (w *FinalizationWorker) finalizeEvents(ctx context.Context, now time.Time) (int, error) {
	var candidates []models.Event
	err := w.db.NewSelect().Model(&candidates).Column("id").
		Where("status = ?", models.EventStatusPublished).Where("completed_at IS NULL").
		Where("end_time + (COALESCE(check_in_grace_minutes, 120) || ' minutes')::interval < ?", now).Scan(ctx)
	if err != nil {
		return 0, fmt.Errorf("list finalization candidates: %w", err)
	}
	finalized := 0
	var failures []error
	for _, candidate := range candidates {
		completed := false
		err := w.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			var event models.Event
			err := tx.NewSelect().Model(&event).Column("id").Where("id = ?", candidate.ID).
				Where("status = ?", models.EventStatusPublished).Where("completed_at IS NULL").
				Where("end_time + (COALESCE(check_in_grace_minutes, 120) || ' minutes')::interval < ?", now).
				For("UPDATE SKIP LOCKED").Scan(ctx)
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			// Attendance and completion commit together; failed attendance remains retryable.
			_, err = tx.ExecContext(ctx, `INSERT INTO event_attendance (registration_id,status,checked_in_at,method)
    SELECT r.id,'attended',?,'auto' FROM event_registrations r
    WHERE r.event_id=? AND r.status='registered' AND EXISTS (
     SELECT 1 FROM event_organizers o WHERE o.event_id=r.event_id AND o.user_id=r.user_id)
    ON CONFLICT (registration_id) DO NOTHING`, now, event.ID)
			if err != nil {
				return fmt.Errorf("auto-attend organizers: %w", err)
			}
			_, err = tx.NewUpdate().Model((*models.Event)(nil)).Set("completed_at = ?", now).Where("id = ?", event.ID).Exec(ctx)
			if err == nil {
				completed = true
			}
			return err
		})
		if err != nil {
			failures = append(failures, fmt.Errorf("finalize event %d: %w", candidate.ID, err))
		} else if completed {
			finalized++
		}
	}
	return finalized, errors.Join(failures...)
}

func (w *FinalizationWorker) archiveOldEvents(ctx context.Context, now time.Time) (int, error) {
	count := 0
	err := w.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var ids []int32
		if err := tx.NewRaw(`UPDATE events SET status='archived',archived_at=?
			WHERE status='published' AND completed_at IS NOT NULL AND end_time < ? RETURNING id`, now, now.AddDate(0, -6, 0)).Scan(ctx, &ids); err != nil {
			return err
		}
		publisher := events.NewDurablePublisher(w.db, nil).InTransaction(tx)
		for _, id := range ids {
			if err := publisher.PublishEventArchived(ctx, events.EventArchivedPayload{EventID: id}); err != nil {
				return err
			}
		}
		count = len(ids)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
