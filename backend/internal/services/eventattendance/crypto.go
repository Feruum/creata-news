package eventattendance

import (
	"time"

	"github.com/studyverse/ems-backend/internal/checkin"
)

const rotationIntervalSeconds = 30

func computeRotatingToken(secret string, windowStart int64) string {
	return checkin.Code(secret, windowStart)
}

func getCurrentWindow() int64 {
	return time.Now().Unix() / rotationIntervalSeconds
}

func getWindowEndTime(window int64) time.Time {
	return time.Unix((window+1)*rotationIntervalSeconds, 0)
}

func validateRotatingToken(secret, providedToken string) bool {
	return checkin.ValidCode(secret, providedToken, time.Now())
}
