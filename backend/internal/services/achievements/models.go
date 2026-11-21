package achievements

import (
	"encoding/json"
	pb "github.com/studyverse/ems-backend/gen/achievements/v1"
	"github.com/uptrace/bun"
	"time"
)

type definition struct {
	bun.BaseModel   `bun:"table:achievements,alias:a"`
	ID              int32 `bun:"id,pk,autoincrement"`
	Title           string
	Description     string
	ImageURL        string `bun:"image_url"`
	BadgeIconURL    string `bun:"badge_icon_url"`
	BackgroundColor string
	AccentColor     string
	Effect          int32
	Status          int32
	Rule            json.RawMessage `bun:"rule,type:jsonb"`
	Revision        int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (d definition) proto(private bool) *pb.Achievement {
	a := &pb.Achievement{Id: d.ID, Title: d.Title, Description: d.Description, ImageUrl: d.ImageURL, BadgeIconUrl: d.BadgeIconURL, BackgroundColor: d.BackgroundColor, AccentColor: d.AccentColor, Effect: pb.AchievementEffect(d.Effect), Status: pb.AchievementStatus(d.Status), CreatedAt: d.CreatedAt.Format(time.RFC3339), UpdatedAt: d.UpdatedAt.Format(time.RFC3339), Revision: d.Revision}
	if private && len(d.Rule) > 0 && string(d.Rule) != "null" {
		a.Rule = &pb.AchievementRule{}
		_ = json.Unmarshal(d.Rule, a.Rule)
	}
	return a
}

type awardState struct {
	Manual     bool
	Automatic  bool
	Suppressed bool
}

func (s awardState) visible() bool { return !s.Suppressed && (s.Manual || s.Automatic) }
func automaticState(old awardState, qualifies bool) awardState {
	old.Automatic = qualifies && !old.Suppressed
	return old
}

type award struct {
	bun.BaseModel  `bun:"table:user_achievements,alias:ua"`
	UserID         int32 `bun:"user_id,pk"`
	AchievementID  int32 `bun:"achievement_id,pk"`
	Manual         bool
	Automatic      bool
	Suppressed     bool
	AwardedAt      time.Time
	Basis          string
	ManualBasis    string
	AutomaticBasis string
	UpdatedAt      time.Time
}
type history struct {
	bun.BaseModel `bun:"table:achievement_history,alias:ah"`
	ID            int64 `bun:"id,pk,autoincrement"`
	AchievementID int32
	UserID        *int32
	ActorID       *int32
	Action        string
	Reason        string
	CreatedAt     time.Time
}
