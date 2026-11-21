package sundayaccount

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/uptrace/bun/dialect/pgdialect"
)

// Onboarding is separate from authentication and may contain only a prompt timestamp.
type Onboarding struct {
	Role        string     `json:"role"`
	RoleOther   string     `json:"roleOther"`
	Interests   []string   `json:"interests" bun:",array"`
	Goals       []string   `json:"goals" bun:",array"`
	Formats     []string   `json:"formats" bun:",array"`
	Source      string     `json:"source"`
	SourceOther string     `json:"sourceOther"`
	PromptedAt  *time.Time `json:"promptedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type onboardingInput struct {
	Role        string   `json:"role"`
	RoleOther   string   `json:"roleOther"`
	Interests   []string `json:"interests"`
	Goals       []string `json:"goals"`
	Formats     []string `json:"formats"`
	Source      string   `json:"source"`
	SourceOther string   `json:"sourceOther"`
}

const onboardingProjection = `role,role_other,interests,goals,formats,source,source_other,prompted_at,completed_at`

func emptyOnboarding() Onboarding {
	return Onboarding{Interests: []string{}, Goals: []string{}, Formats: []string{}}
}

func (h *Handler) getOnboarding(w http.ResponseWriter, r *http.Request, p *principal) {
	onboarding := emptyOnboarding()
	err := h.db.NewRaw(`SELECT `+onboardingProjection+` FROM sunday_open_onboarding WHERE account_id=?`, p.ID).Scan(r.Context(), &onboarding)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fail(w, 503, "Could not load onboarding")
		return
	}
	writeJSON(w, 200, map[string]any{"onboarding": onboarding})
}

func (h *Handler) saveOnboarding(w http.ResponseWriter, r *http.Request, p *principal) {
	var input onboardingInput
	if !decode(w, r, &input) {
		return
	}
	if err := input.validate(); err != nil {
		fail(w, 400, err.Error())
		return
	}
	onboarding := emptyOnboarding()
	err := h.db.NewRaw(`INSERT INTO sunday_open_onboarding(account_id,role,role_other,interests,goals,formats,source,source_other,completed_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,NOW(),NOW())
		ON CONFLICT(account_id) DO UPDATE SET role=EXCLUDED.role,role_other=EXCLUDED.role_other,
			interests=EXCLUDED.interests,goals=EXCLUDED.goals,formats=EXCLUDED.formats,source=EXCLUDED.source,source_other=EXCLUDED.source_other,
			completed_at=COALESCE(sunday_open_onboarding.completed_at,EXCLUDED.completed_at),updated_at=EXCLUDED.updated_at
		RETURNING `+onboardingProjection, p.ID, input.Role, input.RoleOther, pgdialect.Array(input.Interests), pgdialect.Array(input.Goals), pgdialect.Array(input.Formats), input.Source, input.SourceOther).Scan(r.Context(), &onboarding)
	if err != nil {
		fail(w, 503, "Could not save onboarding")
		return
	}
	writeJSON(w, 200, map[string]any{"onboarding": onboarding})
}

func (h *Handler) promptOnboarding(w http.ResponseWriter, r *http.Request, p *principal) {
	onboarding := emptyOnboarding()
	// One upsert makes repeated and concurrent prompts preserve the first timestamp and any answers.
	err := h.db.NewRaw(`INSERT INTO sunday_open_onboarding(account_id,prompted_at,updated_at) VALUES(?,NOW(),NOW())
		ON CONFLICT(account_id) DO UPDATE SET prompted_at=COALESCE(sunday_open_onboarding.prompted_at,EXCLUDED.prompted_at),
			updated_at=CASE WHEN sunday_open_onboarding.prompted_at IS NULL THEN EXCLUDED.updated_at ELSE sunday_open_onboarding.updated_at END
		RETURNING `+onboardingProjection, p.ID).Scan(r.Context(), &onboarding)
	if err != nil {
		fail(w, 503, "Could not mark onboarding prompt")
		return
	}
	writeJSON(w, 200, map[string]any{"onboarding": onboarding})
}

func (input *onboardingInput) validate() error {
	if !onboardingChoice(input.Role, []string{"school_applicant", "parent", "student", "educator", "alumni", "other"}) {
		return errors.New("Choose your role")
	}
	if !onboardingChoice(input.Source, []string{"instagram", "telegram", "tiktok", "youtube", "friends", "school_university", "search", "other"}) {
		return errors.New("Choose how you heard about Sunday Open")
	}
	var err error
	if input.RoleOther, err = onboardingOther(input.Role, input.RoleOther, "role"); err != nil {
		return err
	}
	if input.SourceOther, err = onboardingOther(input.Source, input.SourceOther, "source"); err != nil {
		return err
	}
	if input.Interests, err = onboardingMultiple(input.Interests, []string{"it_ai", "design", "business", "science_engineering", "languages", "creativity", "sports_games", "socializing"}, "interests"); err != nil {
		return err
	}
	if input.Goals, err = onboardingMultiple(input.Goals, []string{"try_new", "practical_skills", "meet_people", "discover_aitu", "choose_career", "find_club"}, "goals"); err != nil {
		return err
	}
	if input.Formats, err = onboardingMultiple(input.Formats, []string{"workshops", "talks", "tours", "club_meetups", "competitions"}, "formats"); err != nil {
		return err
	}
	return nil
}

func onboardingChoice(value string, choices []string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func onboardingOther(choice, raw, field string) (string, error) {
	if choice != "other" {
		return "", nil
	}
	value := strings.TrimSpace(raw)
	if value == "" || utf8.RuneCountInString(value) > 200 || strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("Describe the other %s using 1 to 200 characters", field)
	}
	return value, nil
}

func onboardingMultiple(values, choices []string, field string) ([]string, error) {
	if len(values) == 0 || len(values) > len(choices) {
		return nil, fmt.Errorf("Choose at least one valid %s option", field)
	}
	unique := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if !onboardingChoice(value, choices) {
			return nil, fmt.Errorf("Choose valid %s options", field)
		}
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}
	return unique, nil
}
