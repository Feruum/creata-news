package models

import "time"

func DeriveCheckInPhase(status EventStatus, now, startTime, endTime time.Time, graceMinutes int) CheckInPhase {
	if status != EventStatusPublished {
		return CheckInPhaseNone
	}

	activationTime := startTime.Add(-30 * time.Minute)
	graceEnd := endTime.Add(time.Duration(graceMinutes) * time.Minute)

	switch {
	case now.Before(activationTime):
		return CheckInPhaseNone
	case now.Before(endTime):
		return CheckInPhaseLive
	case now.Before(graceEnd):
		return CheckInPhaseGrace
	default:
		return CheckInPhaseLocked
	}
}
