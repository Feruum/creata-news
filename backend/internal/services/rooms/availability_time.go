package rooms

import (
	"time"
	_ "time/tzdata"
)

func campusAvailabilityDay(value string) (time.Time, error) {
	location, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation("2006-01-02", value, location)
}
