package sundayopen

import (
	"testing"

	pb "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestExternalApprovalStatusesAreVisibleToOrganizers(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   pb.ExternalRegistrationStatus
	}{
		{"pending_approval", pb.ExternalRegistrationStatus_EXTERNAL_REGISTRATION_STATUS_PENDING_APPROVAL},
		{"rejected", pb.ExternalRegistrationStatus_EXTERNAL_REGISTRATION_STATUS_REJECTED},
	} {
		if got := dbExternalStatusToProto(models.ExternalRegistrationStatus(tc.status)); got != tc.want {
			t.Fatalf("%s mapped to %s", tc.status, got)
		}
	}
}
