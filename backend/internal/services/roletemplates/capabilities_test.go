package roletemplates

import (
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"testing"
)

func TestCapabilitiesRejectUnknownAndDeduplicate(t *testing.T) {
	if _, err := validatedCapabilities([]eventsv1.Capability{eventsv1.Capability_CAPABILITY_UNSPECIFIED}); err == nil {
		t.Fatal("unspecified capability accepted")
	}
	if _, err := validatedCapabilities([]eventsv1.Capability{eventsv1.Capability(999)}); err == nil {
		t.Fatal("unknown capability accepted")
	}
	caps, err := validatedCapabilities([]eventsv1.Capability{eventsv1.Capability_CAPABILITY_CREATE_EVENT, eventsv1.Capability_CAPABILITY_CREATE_EVENT})
	if err != nil || len(caps) != 1 {
		t.Fatalf("duplicate grants: %v %v", caps, err)
	}
}
