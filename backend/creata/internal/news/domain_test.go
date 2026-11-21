package news

import "testing"

func TestEditorialPermissions(t *testing.T) {
	a := Doc{"authorId": "author-1", "categoryId": "science", "layer": "world"}
	author := Doc{"id": "author-1", "roles": []any{"author"}}
	if !canEdit(author, a) {
		t.Fatal("author must edit owned draft")
	}
	if canPublish(author, a) {
		t.Fatal("author must not publish")
	}
	editor := Doc{"id": "ed", "roles": []any{"editor"}, "verticals": []any{"science"}}
	if !canPublish(editor, a) {
		t.Fatal("vertical editor must publish world")
	}
	a["layer"] = "eco"
	if canPublish(editor, a) {
		t.Fatal("ecosystem requires chief")
	}
	if canPublish(Doc{"roles": []any{"admin"}}, a) {
		t.Fatal("technical admin must not bypass chief")
	}
}

func TestRevisionInvalidatesApprovalWithoutChangingPublishedSnapshot(t *testing.T) {
	a := Doc{"revision": float64(3), "approvedRevision": float64(3), "publishedRevision": float64(3), "status": "published"}
	revise(a)
	if integer(a["revision"]) != 4 || integer(a["approvedRevision"]) != 0 || integer(a["publishedRevision"]) != 3 || str(a["status"]) != "review" {
		t.Fatalf("wrong revision state: %v", a)
	}
}

func TestCanonicalURL(t *testing.T) {
	a, err := canonicalURL("https://example.org/story?utm_source=test&b=2&a=1#fragment")
	if err != nil || a != "https://example.org/story?a=1&b=2" {
		t.Fatalf("%q %v", a, err)
	}
	if _, err = canonicalURL("javascript:alert(1)"); err == nil {
		t.Fatal("executable URL accepted")
	}
}

func TestNoEmptyLocalePublication(t *testing.T) {
	a := Doc{"title": Doc{"ru": "Есть", "kk": "Бар", "en": "Yes"}, "body": Doc{"ru": "Текст", "kk": "Мәтін", "en": "Text"}, "sources": []any{Doc{"url": "https://example.org"}}}
	if err := publishable(a); err != nil {
		t.Fatal(err)
	}
	a["body"] = Doc{"ru": "Текст", "kk": "", "en": "Text"}
	if err := publishable(a); err == nil {
		t.Fatal("missing Kazakh body accepted")
	}
}
