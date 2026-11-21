package sundayopen

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/studyverse/ems-backend/internal/models"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestExternalRegistrationIncludesOptionalFirstCheckIn(t *testing.T) {
	checked := time.Date(2026, 10, 22, 12, 30, 0, 0, time.UTC)
	for _, timestamp := range []*time.Time{nil, &checked} {
		registration := dbExternalRegistrationToProto(models.ExternalRegistration{ID: 1, EventID: 2, CheckedInAt: timestamp})
		encoded, err := protojson.Marshal(registration)
		if err != nil {
			t.Fatal(err)
		}
		var result struct {
			CheckedInAt *string `json:"checkedInAt"`
		}
		if err := json.Unmarshal(encoded, &result); err != nil {
			t.Fatal(err)
		}
		if (result.CheckedInAt != nil) != (timestamp != nil) {
			t.Fatalf("attendance presence mismatch: %s", encoded)
		}
		if timestamp != nil && *result.CheckedInAt != timestamp.Format(time.RFC3339) {
			t.Fatalf("first attendance timestamp changed: %s", encoded)
		}
	}
}
