package achievements

import (
	pb "github.com/studyverse/ems-backend/gen/achievements/v1"
	"testing"
)

func TestPresentationEffects(t *testing.T) {
	for effect := int32(1); effect <= 5; effect++ {
		a := &pb.Achievement{Title: "Studyverse", Status: pb.AchievementStatus_ACHIEVEMENT_STATUS_ACTIVE, Effect: pb.AchievementEffect(effect), BackgroundColor: "#112233", AccentColor: "#aabbcc"}
		if err := validateDefinition(a); err != nil {
			t.Fatalf("effect %d rejected: %v", effect, err)
		}
	}
	for _, effect := range []int32{0, 6, -1} {
		a := &pb.Achievement{Title: "Studyverse", Status: pb.AchievementStatus_ACHIEVEMENT_STATUS_ACTIVE, Effect: pb.AchievementEffect(effect), BackgroundColor: "#112233", AccentColor: "#aabbcc"}
		if err := validateDefinition(a); err == nil {
			t.Fatalf("invalid effect %d accepted", effect)
		}
	}
}
