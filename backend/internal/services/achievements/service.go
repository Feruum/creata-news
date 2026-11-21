package achievements

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/studyverse/ems-backend/gen/achievements/v1"
	"github.com/studyverse/ems-backend/gen/achievements/v1/achievementsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/uptrace/bun"
)

type Service struct {
	achievementsv1connect.UnimplementedAchievementsServiceHandler
	db      *bun.DB
	authz   *authz.Enforcer
	storage storage.Provider
}

var _ achievementsv1connect.AchievementsServiceHandler = (*Service)(nil)

func NewService(db *bun.DB, enforcer *authz.Enforcer, provider storage.Provider) *Service {
	return &Service{db: db, authz: enforcer, storage: provider}
}
func rpcError(err error) error {
	if err == nil {
		return nil
	}
	var ce *connect.Error
	if errors.As(err, &ce) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return connect.NewError(connect.CodeNotFound, errors.New("achievement not found"))
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("achievements: %w", err))
}
func invalid(message string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(message))
}
func pagination(page, limit int32) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if page > 100000 {
		page = 100000
	}
	return int(limit), int(page-1) * int(limit)
}

func (s *Service) ListAchievements(ctx context.Context, req *connect.Request[pb.ListAchievementsRequest]) (*connect.Response[pb.ListAchievementsResponse], error) {
	if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
		return nil, err
	}
	var defs []definition
	q := s.db.NewSelect().Model(&defs)
	if req.Msg.Search != nil && strings.TrimSpace(*req.Msg.Search) != "" {
		q = q.Where("(a.title ILIKE ? OR a.description ILIKE ?)", "%"+strings.TrimSpace(*req.Msg.Search)+"%", "%"+strings.TrimSpace(*req.Msg.Search)+"%")
	}
	if req.Msg.Status != nil {
		if *req.Msg.Status < 1 || *req.Msg.Status > 3 {
			return nil, invalid("invalid achievement status")
		}
		q = q.Where("a.status = ?", int32(*req.Msg.Status))
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, rpcError(err)
	}
	limit, offset := pagination(req.Msg.Page, req.Msg.Limit)
	if err = q.Order("a.id DESC").Limit(limit).Offset(offset).Scan(ctx); err != nil {
		return nil, rpcError(err)
	}
	out := &pb.ListAchievementsResponse{Total: int32(total)}
	for _, d := range defs {
		out.Achievements = append(out.Achievements, d.proto(true))
	}
	return connect.NewResponse(out), nil
}
func (s *Service) GetAchievement(ctx context.Context, req *connect.Request[pb.GetAchievementRequest]) (*connect.Response[pb.GetAchievementResponse], error) {
	if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
		return nil, err
	}
	var d definition
	if err := s.db.NewSelect().Model(&d).Where("a.id = ?", req.Msg.AchievementId).Scan(ctx); err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&pb.GetAchievementResponse{Achievement: d.proto(true)}), nil
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validateDefinition(a *pb.Achievement) error {
	if a == nil {
		return invalid("achievement is required")
	}
	if a.Id < 0 || len(strings.TrimSpace(a.Title)) == 0 || len(a.Title) > 200 || len(a.Description) > 4000 {
		return invalid("title must contain 1–200 bytes and description at most 4000 bytes")
	}
	if a.Status < 1 || a.Status > 3 || a.Effect < pb.AchievementEffect_ACHIEVEMENT_EFFECT_NONE || a.Effect > pb.AchievementEffect_ACHIEVEMENT_EFFECT_SPARKLE {
		return invalid("a valid status and effect are required")
	}
	if !hexColor.MatchString(a.BackgroundColor) || !hexColor.MatchString(a.AccentColor) {
		return invalid("colors must be six-digit hex colors")
	}
	for _, raw := range []string{a.ImageUrl, a.BadgeIconUrl} {
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || len(raw) > 2048 {
			return invalid("image URLs must use HTTP or HTTPS")
		}
	}
	if r := a.Rule; r != nil {
		if r.Kind < 1 || r.Kind > 4 || r.Threshold < 1 || r.Threshold > 100000 {
			return invalid("rule kind and a positive threshold are required")
		}
		for _, id := range []*int32{r.EventId, r.OrganizationId, r.CategoryId} {
			if id != nil && *id < 1 {
				return invalid("rule filter IDs must be positive")
			}
		}
	}
	return nil
}
func enqueue(ctx context.Context, db bun.IDB, id int32) error {
	_, err := db.NewRaw(`INSERT INTO achievement_recalculation_jobs (achievement_id) VALUES (?) ON CONFLICT (achievement_id) DO UPDATE SET available_at = LEAST(achievement_recalculation_jobs.available_at,NOW())`, id).Exec(ctx)
	return err
}
func record(ctx context.Context, tx bun.IDB, id int32, user, actor *int32, action, reason string) error {
	h := &history{AchievementID: id, UserID: user, ActorID: actor, Action: action, Reason: reason, CreatedAt: time.Now()}
	_, err := tx.NewInsert().Model(h).Exec(ctx)
	return err
}
func lockDefinition(ctx context.Context, tx bun.IDB, id int32) (definition, error) {
	var d definition
	err := tx.NewSelect().Model(&d).Where("a.id = ?", id).For("UPDATE").Scan(ctx)
	return d, err
}
func (s *Service) SaveAchievement(ctx context.Context, req *connect.Request[pb.SaveAchievementRequest]) (*connect.Response[pb.SaveAchievementResponse], error) {
	actor, err := s.authz.RequirePlatformAdmin(ctx)
	if err != nil {
		return nil, err
	}
	a := req.Msg.Achievement
	if err = validateDefinition(a); err != nil {
		return nil, err
	}
	var result definition
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		now := time.Now()
		result = definition{ID: a.Id, Title: strings.TrimSpace(a.Title), Description: a.Description, ImageURL: a.ImageUrl, BadgeIconURL: a.BadgeIconUrl, BackgroundColor: a.BackgroundColor, AccentColor: a.AccentColor, Effect: int32(a.Effect), Status: int32(a.Status), Revision: 1, CreatedAt: now, UpdatedAt: now}
		if a.Rule != nil {
			result.Rule, _ = json.Marshal(a.Rule)
			for table, id := range map[string]*int32{"events": a.Rule.EventId, "organizations": a.Rule.OrganizationId, "categories": a.Rule.CategoryId} {
				if id != nil {
					exists, e := tx.NewSelect().Table(table).Where("id = ?", *id).Exists(ctx)
					if e != nil {
						return e
					}
					if !exists {
						return invalid("rule filter references a missing " + table + " record")
					}
				}
			}
		}
		action := "created"
		if a.Id > 0 {
			old, e := lockDefinition(ctx, tx, a.Id)
			if e != nil {
				return e
			}
			if a.Revision != old.Revision {
				return connect.NewError(connect.CodeAborted, errors.New("achievement changed; reload before saving"))
			}
			result.CreatedAt = old.CreatedAt
			result.Revision = old.Revision + 1
			action = "updated"
			_, err = tx.NewUpdate().Model(&result).WherePK().Exec(ctx)
		} else {
			_, err = tx.NewInsert().Model(&result).Returning("*").Exec(ctx)
		}
		if err != nil {
			return err
		}
		// Removing the name icon or making a definition draft clears its selection.
		if result.BadgeIconURL == "" || a.Status == pb.AchievementStatus_ACHIEVEMENT_STATUS_DRAFT {
			if _, err = tx.NewRaw("DELETE FROM achievement_badge_selections WHERE achievement_id = ?", result.ID).Exec(ctx); err != nil {
				return err
			}
		}
		if err = record(ctx, tx, result.ID, nil, &actor, action, ""); err != nil {
			return err
		}
		return enqueue(ctx, tx, result.ID)
	})
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(&pb.SaveAchievementResponse{Achievement: result.proto(true)}), nil
}
func uniqueUsers(ids []int32) ([]int32, error) {
	if len(ids) == 0 || len(ids) > 500 {
		return nil, invalid("choose between 1 and 500 users")
	}
	seen := map[int32]bool{}
	out := []int32{}
	for _, id := range ids {
		if id < 1 {
			return nil, invalid("user IDs must be positive")
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}
func (s *Service) manualChange(ctx context.Context, id int32, ids []int32, reason string, revoke bool) (int32, error) {
	actor, err := s.authz.RequirePlatformAdmin(ctx)
	if err != nil {
		return 0, err
	}
	ids, err = uniqueUsers(ids)
	if err != nil {
		return 0, err
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 2000 || (revoke && reason == "") {
		return 0, invalid("a revocation reason is required (at most 2000 bytes)")
	}
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		d, e := lockDefinition(ctx, tx, id)
		if e != nil {
			return e
		}
		if !revoke && d.Status != int32(pb.AchievementStatus_ACHIEVEMENT_STATUS_ACTIVE) {
			return invalid("only active achievements can be granted")
		}
		count, e := tx.NewSelect().Table("users").Where("id IN (?)", bun.In(ids)).Count(ctx)
		if e != nil {
			return e
		}
		if count != len(ids) {
			return invalid("one or more users do not exist")
		}
		for _, user := range ids {
			if revoke {
				_, e = tx.NewRaw(`INSERT INTO user_achievements (user_id,achievement_id,suppressed) VALUES (?,?,true) ON CONFLICT (user_id,achievement_id) DO UPDATE SET manual=false,automatic=false,suppressed=true,basis='',manual_basis='',automatic_basis='',updated_at=NOW()`, user, id).Exec(ctx)
			} else {
				_, e = tx.NewRaw(`INSERT INTO user_achievements (user_id,achievement_id,manual,basis,manual_basis) VALUES (?,?,true,?,?) ON CONFLICT (user_id,achievement_id) DO UPDATE SET manual=true,suppressed=false,basis=EXCLUDED.basis,manual_basis=EXCLUDED.manual_basis,awarded_at=CASE WHEN user_achievements.suppressed OR NOT (user_achievements.manual OR user_achievements.automatic) THEN NOW() ELSE user_achievements.awarded_at END,updated_at=NOW()`, user, id, reason, reason).Exec(ctx)
			}
			if e != nil {
				return e
			}
			action := "manual_granted"
			if revoke {
				action = "revoked"
			}
			if e = record(ctx, tx, id, &user, &actor, action, reason); e != nil {
				return e
			}
		}
		return enqueue(ctx, tx, id)
	})
	if err != nil {
		return 0, rpcError(err)
	}
	return int32(len(ids)), nil
}
func (s *Service) GrantAchievement(ctx context.Context, req *connect.Request[pb.GrantAchievementRequest]) (*connect.Response[pb.GrantAchievementResponse], error) {
	n, e := s.manualChange(ctx, req.Msg.AchievementId, req.Msg.UserIds, req.Msg.Reason, false)
	if e != nil {
		return nil, e
	}
	return connect.NewResponse(&pb.GrantAchievementResponse{Affected: n}), nil
}
func (s *Service) RevokeAchievement(ctx context.Context, req *connect.Request[pb.RevokeAchievementRequest]) (*connect.Response[pb.RevokeAchievementResponse], error) {
	n, e := s.manualChange(ctx, req.Msg.AchievementId, req.Msg.UserIds, req.Msg.Reason, true)
	if e != nil {
		return nil, e
	}
	return connect.NewResponse(&pb.RevokeAchievementResponse{Affected: n}), nil
}

func (s *Service) ListAchievementHistory(ctx context.Context, req *connect.Request[pb.ListAchievementHistoryRequest]) (*connect.Response[pb.ListAchievementHistoryResponse], error) {
	if _, e := s.authz.RequirePlatformAdmin(ctx); e != nil {
		return nil, e
	}
	var rows []history
	q := s.db.NewSelect().Model(&rows)
	if req.Msg.AchievementId > 0 {
		q = q.Where("achievement_id = ?", req.Msg.AchievementId)
	}
	if req.Msg.UserId > 0 {
		q = q.Where("user_id = ?", req.Msg.UserId)
	}
	total, e := q.Count(ctx)
	if e != nil {
		return nil, rpcError(e)
	}
	limit, offset := pagination(req.Msg.Page, req.Msg.Limit)
	if e = q.Order("id DESC").Limit(limit).Offset(offset).Scan(ctx); e != nil {
		return nil, rpcError(e)
	}
	out := &pb.ListAchievementHistoryResponse{Total: int32(total)}
	for _, h := range rows {
		entry := &pb.AchievementHistoryEntry{Id: h.ID, AchievementId: h.AchievementID, Action: h.Action, Reason: h.Reason, CreatedAt: h.CreatedAt.Format(time.RFC3339)}
		if h.UserID != nil {
			entry.UserId = *h.UserID
		}
		if h.ActorID != nil {
			entry.ActorId = *h.ActorID
		}
		out.Entries = append(out.Entries, entry)
	}
	return connect.NewResponse(out), nil
}
func (s *Service) targetUser(ctx context.Context, user int32) (int32, error) {
	self, e := s.authz.GetLocalUserID(ctx)
	if e != nil {
		return 0, e
	}
	if user == 0 || user == self {
		return self, nil
	}
	if _, e = s.authz.RequirePlatformAdmin(ctx); e != nil {
		return 0, e
	}
	return user, nil
}
func (s *Service) awards(ctx context.Context, user, id int32, page, limit int32, public bool) ([]*pb.UserAchievement, int, error) {
	var rows []award
	q := s.db.NewSelect().Model(&rows).Join("JOIN achievements AS a ON a.id = ua.achievement_id").Where("ua.user_id = ?", user).Where("NOT ua.suppressed AND (ua.manual OR ua.automatic)").Where("a.status IN (?)", bun.In([]int32{2, 3}))
	if id > 0 {
		q = q.Where("ua.achievement_id = ?", id)
	}
	total, e := q.Count(ctx)
	if e != nil {
		return nil, 0, e
	}
	take, offset := pagination(page, limit)
	if e = q.Order("ua.awarded_at DESC", "ua.achievement_id DESC").Limit(take).Offset(offset).Scan(ctx); e != nil {
		return nil, 0, e
	}
	if len(rows) == 0 {
		return nil, total, nil
	}
	ids := make([]int32, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.AchievementID)
	}
	var defs []definition
	if e = s.db.NewSelect().Model(&defs).Where("a.id IN (?)", bun.In(ids)).Scan(ctx); e != nil {
		return nil, 0, e
	}
	byID := map[int32]definition{}
	for _, d := range defs {
		byID[d.ID] = d
	}
	var selected int32
	e = s.db.NewRaw("SELECT achievement_id FROM achievement_badge_selections WHERE user_id = ?", user).Scan(ctx, &selected)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return nil, 0, e
	}
	out := make([]*pb.UserAchievement, 0, len(rows))
	for _, r := range rows {
		u := &pb.UserAchievement{Achievement: byID[r.AchievementID].proto(!public), UserId: user, AwardedAt: r.AwardedAt.Format(time.RFC3339), Selected: selected == r.AchievementID}
		if !public {
			u.Manual = r.Manual
			u.Automatic = r.Automatic
			u.Basis = r.Basis
		} else if r.Manual {
			u.Basis = "Awarded manually"
		} else {
			u.Basis = r.Basis
		}
		out = append(out, u)
	}
	return out, total, nil
}
func (s *Service) ListUserAchievements(ctx context.Context, req *connect.Request[pb.ListUserAchievementsRequest]) (*connect.Response[pb.ListUserAchievementsResponse], error) {
	if req.Msg.UserId < 0 || req.Msg.AchievementId < 0 {
		return nil, invalid("user and achievement IDs must not be negative")
	}
	user, e := s.targetUser(ctx, req.Msg.UserId)
	if e != nil {
		return nil, e
	}
	rows, total, e := s.awards(ctx, user, req.Msg.AchievementId, req.Msg.Page, req.Msg.Limit, false)
	if e != nil {
		return nil, rpcError(e)
	}
	return connect.NewResponse(&pb.ListUserAchievementsResponse{Achievements: rows, Total: int32(total)}), nil
}
func (s *Service) GetUserAchievement(ctx context.Context, req *connect.Request[pb.GetUserAchievementRequest]) (*connect.Response[pb.GetUserAchievementResponse], error) {
	if req.Msg.UserId < 1 || req.Msg.AchievementId < 1 {
		return nil, invalid("user and achievement IDs are required")
	}
	rows, _, e := s.awards(ctx, req.Msg.UserId, req.Msg.AchievementId, 1, 1, true)
	if e != nil {
		return nil, rpcError(e)
	}
	if len(rows) == 0 {
		return nil, rpcError(sql.ErrNoRows)
	}
	return connect.NewResponse(&pb.GetUserAchievementResponse{UserAchievement: rows[0]}), nil
}
func (s *Service) SetSelectedAchievementBadge(ctx context.Context, req *connect.Request[pb.SetSelectedAchievementBadgeRequest]) (*connect.Response[pb.SetSelectedAchievementBadgeResponse], error) {
	user, e := s.authz.GetLocalUserID(ctx)
	if e != nil {
		return nil, e
	}
	if req.Msg.AchievementId < 0 {
		return nil, invalid("achievement ID must not be negative")
	}
	var badge *pb.NameBadge
	e = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if req.Msg.AchievementId == 0 {
			_, e := tx.NewRaw("DELETE FROM achievement_badge_selections WHERE user_id = ?", user).Exec(ctx)
			return e
		}
		d, e := lockDefinition(ctx, tx, req.Msg.AchievementId)
		if e != nil {
			return e
		}
		if (d.Status != 2 && d.Status != 3) || d.BadgeIconURL == "" {
			return invalid("achievement has no available name badge")
		}
		var a award
		if e = tx.NewSelect().Model(&a).Where("user_id = ? AND achievement_id = ?", user, d.ID).For("UPDATE").Scan(ctx); e != nil {
			return e
		}
		if !(awardState{a.Manual, a.Automatic, a.Suppressed}).visible() {
			return connect.NewError(connect.CodePermissionDenied, errors.New("achievement has not been awarded"))
		}
		if _, e = tx.NewRaw(`INSERT INTO achievement_badge_selections (user_id,achievement_id) VALUES (?,?) ON CONFLICT (user_id) DO UPDATE SET achievement_id=EXCLUDED.achievement_id`, user, d.ID).Exec(ctx); e != nil {
			return e
		}
		badge = &pb.NameBadge{AchievementId: d.ID, Title: d.Title, IconUrl: d.BadgeIconURL, AccentColor: d.AccentColor}
		return nil
	})
	if e != nil {
		return nil, rpcError(e)
	}
	return connect.NewResponse(&pb.SetSelectedAchievementBadgeResponse{SelectedBadge: badge}), nil
}
func (s *Service) RecalculateAchievements(ctx context.Context, req *connect.Request[pb.RecalculateAchievementsRequest]) (*connect.Response[pb.RecalculateAchievementsResponse], error) {
	if _, e := s.authz.RequirePlatformAdmin(ctx); e != nil {
		return nil, e
	}
	if req.Msg.AchievementId < 0 {
		return nil, invalid("invalid achievement ID")
	}
	if req.Msg.AchievementId > 0 {
		exists, e := s.db.NewSelect().Model((*definition)(nil)).Where("id = ?", req.Msg.AchievementId).Exists(ctx)
		if e != nil {
			return nil, rpcError(e)
		}
		if !exists {
			return nil, rpcError(sql.ErrNoRows)
		}
	}
	if e := enqueue(ctx, s.db, req.Msg.AchievementId); e != nil {
		return nil, rpcError(e)
	}
	return connect.NewResponse(&pb.RecalculateAchievementsResponse{Queued: true}), nil
}
func (s *Service) UploadAchievementImage(ctx context.Context, req *connect.Request[pb.UploadAchievementImageRequest]) (*connect.Response[pb.UploadAchievementImageResponse], error) {
	if _, e := s.authz.RequirePlatformAdmin(ctx); e != nil {
		return nil, e
	}
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("image storage is unavailable"))
	}
	if req.Msg.ContentType != "image/png" && req.Msg.ContentType != "image/jpeg" && req.Msg.ContentType != "image/webp" {
		return nil, invalid("image must be PNG, JPEG, or WebP")
	}
	if req.Msg.UploadKind != "image" && req.Msg.UploadKind != "badge" {
		return nil, invalid("upload kind must be image or badge")
	}
	data, e := storage.NormalizeImage(ctx, req.Msg.Image, req.Msg.ContentType)
	if e != nil {
		return nil, invalid(e.Error())
	}
	key := "achievements/" + req.Msg.UploadKind + "/" + uuid.NewString() + ".png"
	if e = s.storage.UploadObject(ctx, key, "image/png", data); e != nil {
		return nil, rpcError(e)
	}
	return connect.NewResponse(&pb.UploadAchievementImageResponse{ImageUrl: s.storage.GetPublicURL(key)}), nil
}
