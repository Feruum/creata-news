package news

import (
	"context"
	"testing"
)

func TestChannelOnlyDoesNotQueuePersonalDigests(t *testing.T) {
	a := setupAPI(t)
	a.app.Config.TelegramChannelOnly = true
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	path := "/admin/articles/" + str(article["id"]) + "/actions"
	a.request("POST", path, Doc{"action": "approve", "expectedRevision": 1}, 200, "")
	a.request("POST", path, Doc{"action": "publish", "expectedRevision": 1}, 200, "")
	a.request("POST", "/internal/bot/users/upsert", Doc{"telegramId": "12345", "locale": "ru"}, 200, a.bot)

	ctx := context.Background()
	settings := DefaultSettings()
	settings["digestHour"] = 0
	if err := a.store.Write(ctx, func(tx *Tx) error { return tx.Put(ctx, "settings", settings) }); err != nil {
		t.Fatal(err)
	}
	if err := a.app.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	deliveries, err := a.store.List(ctx, "delivery")
	if err != nil || len(deliveries) != 0 {
		t.Fatalf("channel-only scheduler queued personal deliveries: %d, err: %v", len(deliveries), err)
	}
	a.request("POST", "/internal/bot/digest", Doc{"telegramId": "12345"}, 409, a.bot)

	a.app.Config.TelegramChannelOnly = false
	if err := a.app.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	deliveries, err = a.store.List(ctx, "delivery")
	if err != nil || len(deliveries) != 1 || str(deliveries[0]["chatId"]) != "12345" {
		t.Fatalf("normal mode did not queue the personal delivery: %v, err: %v", deliveries, err)
	}
}
