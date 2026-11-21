package users

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/studyverse/ems-backend/internal/models"
)

func TestProfileResponsesIncludeStoredAvatar(t *testing.T) {
	var user models.User
	if err := json.Unmarshal([]byte(`{"id":1,"username":"avatar-user","avatar_url":"https://t.me/i/userpic/test.jpg"}`), &user); err != nil {
		t.Fatal(err)
	}
	service := &Service{}
	for name, profile := range map[string]string{
		"current user": service.modelUserToProto(context.Background(), user).GetAvatarUrl(),
		"public user":  modelUserToPublicProto(user).GetAvatarUrl(),
	} {
		if profile != "https://t.me/i/userpic/test.jpg" {
			t.Errorf("%s avatar = %q, want stored photo", name, profile)
		}
	}
}
