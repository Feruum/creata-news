package events

import (
	"testing"

	"github.com/studyverse/ems-backend/internal/models"
)

func TestEventProtoPreservesStoredPublicationTargets(t *testing.T) {
	for _, status := range []models.EventStatus{models.EventStatusDraft, models.EventStatusPublished} {
		for _, targets := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
			event := models.Event{Status: status, PublishOnMap: targets[0], PublishOnSundayOpen: targets[1]}
			proto := dbEventToProto(event, nil, nil)
			if proto.PublishOnMap != targets[0] || proto.PublishOnSundayOpen != targets[1] {
				t.Fatalf("conversion changed stored publication targets %v: %v", targets, proto)
			}
		}
	}
}
