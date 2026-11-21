package services

import (
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"testing"
)

func TestReviewStatusesRemainDistinctInRegistrationProto(t *testing.T) {
	for _, test := range []struct {
		status models.RegistrationStatus
		want   eventsv1.RegistrationStatus
	}{
		{models.RegistrationStatusPendingApproval, eventsv1.RegistrationStatus_REGISTRATION_STATUS_PENDING_APPROVAL},
		{models.RegistrationStatusRejected, eventsv1.RegistrationStatus_REGISTRATION_STATUS_REJECTED},
	} {
		if got := dbRegistrationStatusToProto(test.status); got != test.want {
			t.Fatalf("%s mapped to %v", test.status, got)
		}
	}
}
