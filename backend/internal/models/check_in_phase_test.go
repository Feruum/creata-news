package models

import (
	"testing"
	"time"
)

func TestDeriveCheckInPhase(t *testing.T) {
	base := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)
	startTime := base
	endTime := base.Add(2 * time.Hour)
	graceMinutes := 120

	tests := []struct {
		name   string
		status EventStatus
		now    time.Time
		want   CheckInPhase
	}{
		{
			name:   "draft event returns none",
			status: EventStatusDraft,
			now:    startTime,
			want:   CheckInPhaseNone,
		},
		{
			name:   "cancelled event returns none",
			status: EventStatusCancelled,
			now:    startTime,
			want:   CheckInPhaseNone,
		},
		{
			name:   "archived event returns none",
			status: EventStatusArchived,
			now:    startTime,
			want:   CheckInPhaseNone,
		},
		{
			name:   "well before event returns none",
			status: EventStatusPublished,
			now:    startTime.Add(-2 * time.Hour),
			want:   CheckInPhaseNone,
		},
		{
			name:   "31 minutes before start returns none",
			status: EventStatusPublished,
			now:    startTime.Add(-31 * time.Minute),
			want:   CheckInPhaseNone,
		},
		{
			name:   "30 minutes before start returns live",
			status: EventStatusPublished,
			now:    startTime.Add(-30 * time.Minute),
			want:   CheckInPhaseLive,
		},
		{
			name:   "at start time returns live",
			status: EventStatusPublished,
			now:    startTime,
			want:   CheckInPhaseLive,
		},
		{
			name:   "during event returns live",
			status: EventStatusPublished,
			now:    startTime.Add(1 * time.Hour),
			want:   CheckInPhaseLive,
		},
		{
			name:   "1 second before end returns live",
			status: EventStatusPublished,
			now:    endTime.Add(-1 * time.Second),
			want:   CheckInPhaseLive,
		},
		{
			name:   "at end time returns grace",
			status: EventStatusPublished,
			now:    endTime,
			want:   CheckInPhaseGrace,
		},
		{
			name:   "during grace period returns grace",
			status: EventStatusPublished,
			now:    endTime.Add(1 * time.Hour),
			want:   CheckInPhaseGrace,
		},
		{
			name:   "at grace end returns locked",
			status: EventStatusPublished,
			now:    endTime.Add(time.Duration(graceMinutes) * time.Minute),
			want:   CheckInPhaseLocked,
		},
		{
			name:   "after grace end returns locked",
			status: EventStatusPublished,
			now:    endTime.Add(3 * time.Hour),
			want:   CheckInPhaseLocked,
		},
		{
			name:   "zero grace minutes locks at end time",
			status: EventStatusPublished,
			now:    endTime,
			want:   CheckInPhaseLocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grace := graceMinutes
			if tt.name == "zero grace minutes locks at end time" {
				grace = 0
			}
			got := DeriveCheckInPhase(tt.status, tt.now, startTime, endTime, grace)
			if got != tt.want {
				t.Errorf("DeriveCheckInPhase() = %q, want %q", got, tt.want)
			}
		})
	}
}
