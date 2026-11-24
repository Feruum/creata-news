package news

import (
	"strings"
	"testing"
)

func TestUnconfiguredChannelDeliveryLeavesDigestApproved(t *testing.T) {
	a := setupAPI(t)
	a.login("chief_editor")
	a.app.Config.ChannelID = ""

	draft := a.request("POST", "/admin/digests/preview", Doc{"locale": "ru"}, 201, "")
	path := "/admin/digests/" + str(draft["id"])
	a.request("POST", path+"/approve", nil, 200, "")
	list := a.request("GET", "/admin/digests", nil, 200, "")
	if str(list["channelDeliveryStatus"]) != "channel_missing" {
		t.Fatalf("missing channel should be visible in digest settings: %v", list["channelDeliveryStatus"])
	}
	reply := a.request("POST", path+"/send", nil, 503, "")
	if str(reply["error"]) != "Telegram channel is not configured" {
		t.Fatalf("unexpected delivery rejection: %v", reply)
	}

	a.app.Config.ChannelID = "-1001234"
	a.app.Config.TelegramBotUnconfigured = true
	list = a.request("GET", "/admin/digests", nil, 200, "")
	if str(list["channelDeliveryStatus"]) != "bot_missing" {
		t.Fatalf("missing bot should be visible in digest settings: %v", list["channelDeliveryStatus"])
	}
	reply = a.request("POST", path+"/send", nil, 503, "")
	if str(reply["error"]) != "Telegram bot is not configured" {
		t.Fatalf("unexpected bot rejection: %v", reply)
	}

	list = a.request("GET", "/admin/digests", nil, 200, "")
	if str(object(array(list["items"])[0])["status"]) != "approved" {
		t.Fatal("the rejected delivery changed the approved edition")
	}
	if entries := array(a.request("GET", "/admin/deliveries", nil, 200, "")["items"]); len(entries) != 0 {
		t.Fatalf("unexpected delivery records: %v", entries)
	}
}

func TestDigestArticleLinkEscapesAnImportedSlug(t *testing.T) {
	s := &Server{Config: Config{PublicURL: "https://news.example.org/"}}
	article := Doc{"slug": "за туркестанской-рысью"}
	link := s.articleLink(article, "ru")
	if link != "https://news.example.org/ru/articles/%D0%B7%D0%B0%20%D1%82%D1%83%D1%80%D0%BA%D0%B5%D1%81%D1%82%D0%B0%D0%BD%D1%81%D0%BA%D0%BE%D0%B9-%D1%80%D1%8B%D1%81%D1%8C%D1%8E" {
		t.Fatalf("unexpected article link: %q", link)
	}
	d := Doc{"locale": "ru", "articles": []any{article}}
	if !strings.Contains(s.digestText(d), "\n"+link+"\n") {
		t.Fatal("digest text did not include the escaped article link")
	}
}
