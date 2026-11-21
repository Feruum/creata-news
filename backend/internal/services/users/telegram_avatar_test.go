//go:build integration

package users

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
)

func TestLinkTelegramPersistsVerifiedAvatar(t *testing.T) {
	db := newTestDB(t)
	ctx := authedContext("avatar-test")
	userID := seedAuthUser(t, db, "avatar-test")
	const token = "synthetic-avatar-test-token"
	const photo = "https://t.me/i/userpic/test.jpg"
	svc := NewService(db, nil, nil, nil, nil, "", token)

	link := func(photoURL string) *usersv1.LinkTelegramRequest {
		t.Helper()
		req := &usersv1.LinkTelegramRequest{TelegramId: 123456, FirstName: "QA", PhotoUrl: photoURL, AuthDate: time.Now().Unix()}
		check := fmt.Sprintf("auth_date=%d\nfirst_name=QA\nid=123456", req.AuthDate)
		if photoURL != "" {
			check += "\nphoto_url=" + photoURL
		}
		key := sha256.Sum256([]byte(token))
		mac := hmac.New(sha256.New, key[:])
		_, _ = mac.Write([]byte(check))
		req.Hash = hex.EncodeToString(mac.Sum(nil))
		return req
	}

	response, err := svc.LinkTelegram(ctx, connect.NewRequest(link(photo)))
	if err != nil {
		t.Fatalf("link Telegram: %v", err)
	}
	if response.Msg.User.GetAvatarUrl() != photo {
		t.Fatalf("linked avatar = %q", response.Msg.User.GetAvatarUrl())
	}
	var stored string
	if err := db.NewRaw("SELECT avatar_url FROM users WHERE id = ?", userID).Scan(ctx, &stored); err != nil || stored != photo {
		t.Fatalf("stored avatar = %q, error = %v", stored, err)
	}
	for range 2 {
		profile, err := svc.GetMe(ctx, connect.NewRequest(&usersv1.GetMeRequest{}))
		if err != nil || profile.Msg.User.GetAvatarUrl() != photo {
			t.Fatalf("profile reload lost linked avatar: response=%v error=%v", profile, err)
		}
	}

	response, err = svc.LinkTelegram(ctx, connect.NewRequest(link("")))
	if err != nil || response.Msg.User.GetAvatarUrl() != photo {
		t.Fatalf("empty photo relink lost avatar: response=%v error=%v", response, err)
	}
	const customPhoto = "https://example.com/my-avatar.jpg"
	if _, err := db.NewRaw("UPDATE users SET avatar_url = ? WHERE id = ?", customPhoto, userID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	response, err = svc.LinkTelegram(ctx, connect.NewRequest(link(photo)))
	if err != nil || response.Msg.User.GetAvatarUrl() != customPhoto {
		t.Fatalf("custom avatar overwritten: response=%v error=%v", response, err)
	}
	invalid := link(photo)
	invalid.PhotoUrl = "https://example.com/tampered.jpg"
	if _, err := svc.LinkTelegram(ctx, connect.NewRequest(invalid)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("tampered photo was not rejected: %v", err)
	}
}
