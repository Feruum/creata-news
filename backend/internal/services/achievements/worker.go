package achievements

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	pb "github.com/studyverse/ems-backend/gen/achievements/v1"
	"github.com/uptrace/bun"
	"log/slog"
	"time"
)

// Reconcile always derives qualification from the current database. Definition
// locks serialize it with edits, manual overrides, and badge selection.
func Reconcile(ctx context.Context, db *bun.DB, id int32) error {
	var defs []definition
	q := db.NewSelect().Model(&defs).Order("a.id")
	if id > 0 {
		q = q.Where("a.id = ?", id)
	}
	if err := q.Scan(ctx); err != nil {
		return fmt.Errorf("list achievement definitions: %w", err)
	}
	for _, candidate := range defs {
		if err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			d, err := lockDefinition(ctx, tx, candidate.ID)
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			if d.Status != int32(pb.AchievementStatus_ACHIEVEMENT_STATUS_ACTIVE) {
				return nil
			}
			qualified := map[int32]bool{}
			var rule pb.AchievementRule
			if len(d.Rule) > 0 && string(d.Rule) != "null" {
				if err = json.Unmarshal(d.Rule, &rule); err != nil {
					return fmt.Errorf("decode rule %d: %w", d.ID, err)
				}
				if rule.Kind < 1 || rule.Kind > 4 || rule.Threshold < 1 {
					return fmt.Errorf("invalid stored achievement rule %d", d.ID)
				}
				var users []int32
				query := tx.NewSelect().TableExpr("event_registrations AS er").ColumnExpr("er.user_id").Join("JOIN events AS e ON e.id = er.event_id").Where("er.status = 'registered'").Where("e.status <> 'cancelled'")
				if rule.EventId != nil {
					query = query.Where("e.id = ?", *rule.EventId)
				}
				if rule.OrganizationId != nil {
					query = query.Where("e.organization_id = ?", *rule.OrganizationId)
				}
				if rule.CategoryId != nil {
					query = query.Where("e.category_id = ?", *rule.CategoryId)
				}
				if rule.Kind != pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_REGISTRATIONS {
					query = query.Where(`EXISTS (SELECT 1 FROM event_attendance ea WHERE ea.registration_id=er.id AND ea.status IN ('checked_in','attended'))`)
				}
				if rule.Kind == pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_SOC_GPA_ATTENDANCE {
					query = query.Where("e.soc_gpa_badge_status = 'approved'")
				}
				if rule.Kind == pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_IROS_ATTENDANCE {
					query = query.Where("e.iros_badge_status = 'approved'")
				}
				if err = query.Group("er.user_id").Having("COUNT(DISTINCT e.id) >= ?", rule.Threshold).Scan(ctx, &users); err != nil {
					return err
				}
				for _, user := range users {
					qualified[user] = true
				}
			}
			var awards []award
			if err = tx.NewSelect().Model(&awards).Where("achievement_id = ?", d.ID).For("UPDATE").Scan(ctx); err != nil {
				return err
			}
			existing := map[int32]bool{}
			basis := ruleBasis(&rule)
			for _, a := range awards {
				existing[a.UserID] = true
				old := awardState{a.Manual, a.Automatic, a.Suppressed}
				next := automaticState(old, qualified[a.UserID])
				automaticBasis := ""
				if next.Automatic {
					automaticBasis = basis
				}
				displayBasis := automaticBasis
				if a.Manual {
					displayBasis = a.ManualBasis
				}
				if old == next && a.AutomaticBasis == automaticBasis && a.Basis == displayBasis {
					continue
				}
				a.Automatic = next.Automatic
				a.UpdatedAt = time.Now()
				if !old.visible() && next.visible() {
					a.AwardedAt = a.UpdatedAt
				}
				a.AutomaticBasis = automaticBasis
				a.Basis = displayBasis
				if _, err = tx.NewUpdate().Model(&a).WherePK().Exec(ctx); err != nil {
					return err
				}
				if old == next {
					continue
				}
				action := "automatic_removed"
				if next.Automatic {
					action = "automatic_granted"
				}
				if err = record(ctx, tx, d.ID, &a.UserID, nil, action, basis); err != nil {
					return err
				}
			}
			for user := range qualified {
				if existing[user] {
					continue
				}
				a := &award{UserID: user, AchievementID: d.ID, Automatic: true, AwardedAt: time.Now(), UpdatedAt: time.Now(), Basis: basis, AutomaticBasis: basis}
				if _, err = tx.NewInsert().Model(a).Exec(ctx); err != nil {
					return err
				}
				if err = record(ctx, tx, d.ID, &user, nil, "automatic_granted", basis); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("reconcile achievement %d: %w", candidate.ID, err)
		}
	}
	return nil
}

func ruleBasis(rule *pb.AchievementRule) string {
	events := "events"
	if rule.Threshold == 1 {
		events = "event"
	}
	switch rule.Kind {
	case pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_REGISTRATIONS:
		return fmt.Sprintf("Registered for %d qualifying %s", rule.Threshold, events)
	case pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_ATTENDANCE:
		return fmt.Sprintf("Confirmed attendance at %d qualifying %s", rule.Threshold, events)
	case pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_SOC_GPA_ATTENDANCE:
		return fmt.Sprintf("Confirmed attendance at %d approved Soc GPA %s", rule.Threshold, events)
	case pb.AchievementRuleKind_ACHIEVEMENT_RULE_KIND_IROS_ATTENDANCE:
		return fmt.Sprintf("Confirmed attendance at %d approved iROS %s", rule.Threshold, events)
	default:
		return "Automatic rule removed"
	}
}

type Worker struct{ db *bun.DB }

func NewWorker(db *bun.DB) *Worker { return &Worker{db: db} }
func (w *Worker) Start(ctx context.Context) {
	if err := enqueue(ctx, w.db, 0); err != nil {
		slog.Error("enqueue achievement reconciliation", "error", err)
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	periodic := time.NewTicker(time.Minute)
	defer periodic.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-periodic.C:
			if err := enqueue(ctx, w.db, 0); err != nil {
				slog.Error("enqueue periodic achievements", "error", err)
			}
		case <-tick.C:
			if err := w.ProcessNext(ctx); err != nil {
				slog.Error("achievement worker", "error", err)
			}
		}
	}
}

// Claim a renewable lease in a short transaction. Never hold the queue lock
// while locking definitions: source writes and admin edits enqueue in their own
// transactions. The lease predicate preserves any job requeued during a run.
func (w *Worker) ProcessNext(ctx context.Context) error {
	var job struct {
		AchievementID int32
		Attempts      int
		AvailableAt   time.Time
	}
	err := w.db.NewRaw(`UPDATE achievement_recalculation_jobs SET attempts=attempts+1, available_at=NOW()+INTERVAL '5 minutes'
 WHERE achievement_id=(SELECT achievement_id FROM achievement_recalculation_jobs WHERE available_at<=NOW() ORDER BY available_at,achievement_id FOR UPDATE SKIP LOCKED LIMIT 1)
 RETURNING achievement_id,attempts,available_at`).Scan(ctx, &job)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	runCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	if job.AchievementID == 0 {
		// Fan out the full-history request durably. Each definition then gets
		// its own lease, timeout and retry budget, so a large or failing first
		// definition cannot starve later definitions on the global scan.
		err = w.db.RunInTx(runCtx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.NewRaw(`INSERT INTO achievement_recalculation_jobs (achievement_id)
				SELECT id FROM achievements WHERE status=2 ORDER BY id
				ON CONFLICT (achievement_id) DO UPDATE SET available_at=LEAST(achievement_recalculation_jobs.available_at,NOW())`).Exec(ctx); err != nil {
				return err
			}
			_, err := tx.NewRaw(`DELETE FROM achievement_recalculation_jobs WHERE achievement_id=0 AND available_at=?`, job.AvailableAt).Exec(ctx)
			return err
		})
	} else {
		err = Reconcile(runCtx, w.db, job.AchievementID)
	}
	if err != nil {
		delay := time.Duration(min(job.Attempts, 60)) * time.Second
		_, updateErr := w.db.NewRaw(`UPDATE achievement_recalculation_jobs SET available_at=?,last_error=? WHERE achievement_id=? AND available_at=?`, time.Now().Add(delay), err.Error(), job.AchievementID, job.AvailableAt).Exec(ctx)
		if updateErr != nil {
			return fmt.Errorf("reconcile failed: %v; retry schedule: %w", err, updateErr)
		}
		return err
	}
	_, err = w.db.NewRaw(`DELETE FROM achievement_recalculation_jobs WHERE achievement_id=? AND available_at=?`, job.AchievementID, job.AvailableAt).Exec(ctx)
	return err
}
