package news

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const defaultMorningDigestHour = 9

type digestSelection struct {
	now           time.Time
	morningCutoff time.Time
}

func isCentralAsianRegion(region any) bool {
	switch strings.ToLower(strings.TrimSpace(str(region))) {
	case "central-asia", "central asia", "центральная азия", "орталық азия",
		"kz", "kazakhstan", "казахстан", "қазақстан",
		"kg", "kyrgyzstan", "кыргызстан", "қырғызстан",
		"tj", "tajikistan", "таджикистан", "тәжікстан",
		"tm", "turkmenistan", "туркменистан", "түрікменстан",
		"uz", "uzbekistan", "узбекистан", "өзбекстан":
		return true
	default:
		return false
	}
}

func morningDigestHour(settings Doc) int {
	if configured, ok := settings["morningDigestHour"]; ok {
		return integer(configured)
	}
	return defaultMorningDigestHour
}

func (s *Server) scheduleMorningDigest(t *Tx, ctx context.Context, current time.Time, settings Doc) error {
	cutoff := time.Date(current.Year(), current.Month(), current.Day(), morningDigestHour(settings), 0, 0, 0, current.Location())
	if current.Before(cutoff) {
		return nil
	}
	day := current.Format(time.DateOnly)
	id := "morning-" + day
	var exists bool
	if err := t.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM creata_records WHERE kind='digest' AND id=$1)", id).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	draft, err := s.compose(t, ctx, Doc{"id": "channel", "locale": "ru"}, "channel", digestSelection{
		now: current, morningCutoff: cutoff,
	})
	if err != nil {
		return err
	}
	// The date-keyed record is the edition and the durable deduplication marker.
	// Tick holds the shared transaction lock for the lookup and insert.
	draft["id"] = id
	return t.Put(ctx, "digest", draft)
}

func (s *Server) morningDigestStatus(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, true); err != nil {
		return err
	}
	var edition Doc
	var collectedAt *time.Time
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT
		(SELECT jsonb_build_object('id', id, 'date', data->>'morningDate', 'status', data->>'status',
			'articleCount', jsonb_array_length(data->'articles'),
			'opportunityCount', jsonb_array_length(data->'opportunities'))
		FROM creata_records WHERE kind='digest' AND id LIKE 'morning-%' ORDER BY id DESC LIMIT 1),
		(SELECT max(updated_at) FROM creata_records
		WHERE kind='job' AND data->>'kind'='collect' AND data->>'status'='succeeded')`).Scan(&edition, &collectedAt)
	if err != nil {
		return err
	}
	out := Doc{}
	if edition != nil {
		out["edition"] = edition
	}
	if collectedAt != nil {
		out["lastCollectionAt"] = collectedAt.UTC().Format(time.RFC3339Nano)
	}
	respond(w, 200, out)
	return nil
}
