package rooms

import (
	"testing"
	"time"
)

func TestAvailabilityDayUsesCampusTimezone(t *testing.T) {
	day, err := campusAvailabilityDay("2026-10-04")
	if err != nil {
		t.Fatal(err)
	}
	if got := day.UTC().Format(time.RFC3339); got != "2026-10-03T19:00:00Z" {
		t.Fatalf("day start = %s", got)
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), 8, 0, 0, 0, day.Location())
	if got := start.Format(time.RFC3339); got != "2026-10-04T08:00:00+05:00" {
		t.Fatalf("first slot = %s", got)
	}
	if _, err := campusAvailabilityDay("2026-02-30"); err == nil {
		t.Fatal("invalid date accepted")
	}
}
