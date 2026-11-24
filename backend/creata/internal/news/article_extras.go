package news

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var (
	storyUpdateID    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,79}$`)
	storyArticleSlug = regexp.MustCompile(`^[^\p{Cc}/\\?#]{1,120}$`)
	storyUpdateTime  = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)
)

func (t *Tx) validateStoryPermalink(ctx context.Context, article Doc) error {
	slug := str(article["slug"])
	if established := str(article["storySlug"]); established != "" {
		if slug != established {
			return errStatus(409, "A published story's slug cannot change")
		}
		return nil
	}
	if len(array(article["storyUpdates"])) == 0 {
		return nil
	}
	// On conversion, the live slug may differ from the latest draft slug.
	var publishedSlug string
	err := t.QueryRow(ctx, "SELECT data->>'slug' FROM creata_records WHERE kind='publication' AND id=$1", str(article["id"])).Scan(&publishedSlug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if slug != publishedSlug {
		return errStatus(409, fmt.Sprintf("Keep the published article slug %q when adding story updates", publishedSlug))
	}
	return nil
}

func validateArticleExtras(article Doc, publication bool) error {
	if photos, exists := article["photos"]; exists {
		if err := validateArticlePhotos(photos); err != nil {
			return err
		}
	}
	if value := article["actionGuide"]; value != nil {
		guide := object(value)
		if len(guide) != 3 {
			return fmt.Errorf("actionGuide must contain audience, steps and sourceUrl")
		}
		if err := validateArticleExtra(guide, "audience", "steps", publication); err != nil {
			return fmt.Errorf("actionGuide: %w", err)
		}
	}
	if value, exists := article["storyUpdates"]; exists {
		updates, ok := value.([]any)
		if !ok || len(updates) > 40 {
			return fmt.Errorf("storyUpdates must be an array of up to 40 updates")
		}
		seenIDs := map[string]bool{}
		for index, value := range updates {
			if err := validateStoryUpdate(object(value), seenIDs, publication); err != nil {
				return fmt.Errorf("storyUpdates[%d]: %w", index, err)
			}
		}
	}
	return nil
}

func validateStoryUpdate(update Doc, seenIDs map[string]bool, publication bool) error {
	fieldCount := 5
	if slug, exists := update["articleSlug"]; exists {
		value := str(slug)
		if value == "." || value == ".." || strings.TrimSpace(value) != value || !storyArticleSlug.MatchString(value) {
			return fmt.Errorf("articleSlug must be a valid article slug")
		}
		fieldCount++
	}
	if len(update) != fieldCount {
		return fmt.Errorf("update must contain id, occurredAt, title, body and sourceUrl")
	}
	updateID := str(update["id"])
	if !storyUpdateID.MatchString(updateID) || seenIDs[updateID] {
		return fmt.Errorf("id must be a unique update identifier")
	}
	seenIDs[updateID] = true
	occurredAt := str(update["occurredAt"])
	if !storyUpdateTime.MatchString(occurredAt) {
		return fmt.Errorf("occurredAt must be an RFC3339 date and time")
	}
	if _, err := time.Parse(time.RFC3339Nano, occurredAt); err != nil {
		return fmt.Errorf("occurredAt must be an RFC3339 date and time")
	}
	return validateArticleExtra(update, "title", "body", publication)
}

func validateArticleExtra(extra Doc, shortField, longField string, publication bool) error {
	shortText, longText := object(extra[shortField]), object(extra[longField])
	if err := validateExtraText(shortText, shortField, 300); err != nil {
		return err
	}
	if err := validateExtraText(longText, longField, 4000); err != nil {
		return err
	}
	source, ok := extra["sourceUrl"].(string)
	if !ok || utf8.RuneCountInString(source) > 2048 {
		return fmt.Errorf("sourceUrl must be a string of up to 2048 characters")
	}
	source = strings.TrimSpace(source)
	if source != "" && !validArticleSourceURL(source) {
		return fmt.Errorf("sourceUrl must be an absolute HTTP or HTTPS URL without credentials")
	}
	if publication {
		if source == "" {
			return fmt.Errorf("sourceUrl is required for publication")
		}
		for _, lang := range []string{"ru", "kk", "en"} {
			if strings.TrimSpace(str(shortText[lang])) != "" && strings.TrimSpace(str(longText[lang])) != "" {
				return nil
			}
		}
		return fmt.Errorf("%s and %s are required in at least one shared language", shortField, longField)
	}
	return nil
}

func validArticleSourceURL(raw string) bool {
	parsed, err := parsePublicURL(raw)
	if err != nil || strings.ContainsRune(raw, '\\') {
		return false
	}
	if port := parsed.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return false
		}
	}
	if strings.HasPrefix(parsed.Host, "[") {
		ip, err := netip.ParseAddr(parsed.Hostname())
		return err == nil && ip.Is6() && ip.Zone() == ""
	}
	return true
}

func validateExtraText(localized Doc, field string, limit int) error {
	if len(localized) != 3 {
		return fmt.Errorf("%s must contain text in ru, kk and en", field)
	}
	for _, lang := range []string{"ru", "kk", "en"} {
		if value, ok := localized[lang].(string); !ok || strings.ContainsRune(value, 0) || utf8.RuneCountInString(value) > limit {
			return fmt.Errorf("%s.%s must be text of up to %d characters", field, lang, limit)
		}
	}
	return nil
}

func projectArticleExtras(article Doc) {
	if guide := object(article["actionGuide"]); len(guide) > 0 {
		article["actionGuide"] = Doc{
			"audience":  projectExtraText(guide["audience"]),
			"steps":     projectExtraText(guide["steps"]),
			"sourceUrl": strings.TrimSpace(str(guide["sourceUrl"])),
		}
	} else {
		delete(article, "actionGuide")
	}
	updates := array(article["storyUpdates"])
	if len(updates) == 0 {
		delete(article, "storyUpdates")
		return
	}
	projected := make([]any, 0, len(updates))
	for _, raw := range updates {
		update := object(raw)
		public := Doc{
			"id": update["id"], "occurredAt": update["occurredAt"],
			"title": projectExtraText(update["title"]), "body": projectExtraText(update["body"]),
			"sourceUrl": strings.TrimSpace(str(update["sourceUrl"])),
		}
		if slug := str(update["articleSlug"]); slug != "" {
			public["articleSlug"] = slug
		}
		projected = append(projected, public)
	}
	sort.SliceStable(projected, func(i, j int) bool {
		return parseTime(object(projected[i])["occurredAt"]).After(parseTime(object(projected[j])["occurredAt"]))
	})
	article["storyUpdates"] = projected
}

func projectExtraText(text any) Doc {
	localized := object(text)
	return Doc{"ru": str(localized["ru"]), "kk": str(localized["kk"]), "en": str(localized["en"])}
}
