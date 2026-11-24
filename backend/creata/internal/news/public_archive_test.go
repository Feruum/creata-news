package news

import (
	"context"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPublicArticleArchivePagesAndPreservesSearchFallback(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		if err := tx.Put(ctx, "author", Doc{"id": "writer", "name": "Archive Writer"}); err != nil {
			return err
		}
		for _, record := range []Doc{
			{"id": "c", "format": "article", "categoryId": "ai"},
			{"id": "b-other-category", "categoryId": "science"},
			{"id": "b-other-layer", "categoryId": "ai", "layer": "world"},
			{"id": "b-other-author", "categoryId": "ai", "authorId": "outside"},
			{"id": "b-other-region", "categoryId": "ai", "region": "Астана"},
			{"id": "a", "categoryId": "ai", "title": Doc{"ru": "СЮЖЕТ", "kk": "ҚАЗАҚСТАН"}},
			{"id": "b-graphic", "format": "graphic", "categoryId": "ai"},
		} {
			article := Doc{
				"slug": record["id"], "layer": "eco", "region": "Алматы", "authorId": "writer", "status": "published",
				"body":  Doc{"ru": strings.Repeat("слово\u00a0", 221) + "буквально 100%_ФАКТ"},
				"title": Doc{"ru": "ЗАГОЛОВОК", "kk": "\u00a0"},
			}
			for key, value := range record {
				article[key] = value
			}
			if err := tx.Put(ctx, "publication", article); err != nil {
				return err
			}
		}
		// Every insertion in this transaction has the same created_at.
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	query := "/public/articles?format=article&category=ai&layer=eco&authorId=writer&region=" + url.QueryEscape("Алматы") +
		"&locale=kk&q=" + url.QueryEscape("100%_факт")
	summaryQuery := query + "&limit=1&view=summary"
	first := a.request("GET", summaryQuery, nil, 200, "")
	firstItems := array(first["items"])
	if len(firstItems) != 1 || str(object(firstItems[0])["id"]) != "a" || integer(first["total"]) != 2 {
		t.Fatalf("unexpected first page length %d or total %v", len(firstItems), first["total"])
	}
	firstArticle := object(firstItems[0])
	if _, hasBody := firstArticle["body"]; hasBody {
		t.Fatal("a summary included the article body")
	}
	if integer(object(firstArticle["readingMinutes"])["kk"]) != 2 {
		t.Fatal("a summary did not estimate reading time for the fallback Russian text")
	}
	if str(object(firstArticle["author"])["name"]) != "Archive Writer" {
		t.Fatal("a summary did not include the publication's author")
	}
	cursor := str(first["nextCursor"])
	if cursor == "" {
		t.Fatal("first page is missing a continuation cursor")
	}
	second := a.request("GET", summaryQuery+"&cursor="+url.QueryEscape(cursor), nil, 200, "")
	secondItems := array(second["items"])
	if len(secondItems) != 1 || str(object(secondItems[0])["id"]) != "c" || integer(second["total"]) != 2 || second["nextCursor"] != nil {
		t.Fatalf("unexpected second page: %v", second)
	}
	translated := a.request("GET", query+"&limit=1&translated=1", nil, 200, "")
	translatedItems := array(translated["items"])
	if len(translatedItems) != 1 || str(object(translatedItems[0])["id"]) != "a" || translated["nextCursor"] != nil {
		t.Fatalf("unexpected page with translated titles: %v", translated)
	}
	legacy := a.request("GET", query, nil, 200, "")
	legacyItems := array(legacy["items"])
	if len(legacyItems) != 2 || legacy["nextCursor"] != nil || localize(object(legacyItems[0])["body"], "kk") == "" {
		t.Fatalf("unexpected legacy response length %d or cursor %v", len(legacyItems), legacy["nextCursor"])
	}
	withoutTotal := a.request("GET", summaryQuery+"&includeTotal=0", nil, 200, "")
	if len(array(withoutTotal["items"])) != 1 || withoutTotal["total"] != nil || withoutTotal["nextCursor"] != cursor {
		t.Fatalf("unexpected page without total: %v", withoutTotal)
	}
	noMatches := a.request("GET", "/public/articles?limit=1&view=summary&category=missing", nil, 200, "")
	if len(array(noMatches["items"])) != 0 || noMatches["total"] != float64(0) || noMatches["nextCursor"] != nil {
		t.Fatalf("unexpected empty page: %v", noMatches)
	}
	kazakh := a.request("GET", "/public/articles?locale=kk&q="+url.QueryEscape("қазақстан"), nil, 200, "")
	kazakhItems := array(kazakh["items"])
	if len(kazakhItems) != 1 || str(object(kazakhItems[0])["id"]) != "a" {
		t.Fatalf("Kazakh search returned %d matches", len(kazakhItems))
	}
}

func TestPublicArticleArchiveRejectsInvalidPagination(t *testing.T) {
	a := setupAPI(t)
	for _, query := range []string{"limit=0", "limit=101", "limit=abc", "limit=1&cursor=wrong", "cursor=wrong"} {
		a.request("GET", "/public/articles?"+query, nil, 400, "")
	}
}

func TestPublicArticleUsesSnapshotSlugAndOwnCorrections(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		for _, record := range []struct {
			kind string
			doc  Doc
		}{
			{"article", Doc{"id": "article-a", "slug": "unpublished-slug", "title": localized("Draft")}},
			{"publication", Doc{"id": "article-a", "slug": "published-slug", "title": localized("Snapshot"), "revision": 3}},
			{"publication_version", Doc{"id": "article-a:1", "articleId": "article-a", "revision": 1, "note": ""}},
			{"publication_version", Doc{"id": "article-a:2", "articleId": "article-a", "revision": 2, "note": "Updated attribution", "publishedAt": "2026-11-22T00:00:00Z"}},
			{"publication_version", Doc{"id": "article-b:2", "articleId": "article-b", "revision": 2, "note": "Unrelated correction"}},
		} {
			if err := tx.Put(ctx, record.kind, record.doc); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	detail := a.request("GET", "/public/articles/published-slug", nil, 200, "")
	corrections := array(detail["corrections"])
	if localize(detail["title"], "en") != "Snapshot" || len(corrections) != 1 || str(object(corrections[0])["note"]) != "Updated attribution" {
		t.Fatalf("wrong snapshot or corrections: %v", detail)
	}
	a.request("GET", "/public/articles/unpublished-slug", nil, 404, "")
	if err := a.store.Write(ctx, func(tx *Tx) error { return tx.Delete(ctx, "publication", "article-a") }); err != nil {
		t.Fatal(err)
	}
	a.request("GET", "/public/articles/published-slug", nil, 404, "")
}

func TestPublicRegionsDeduplicatesPublishedArticleRegions(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		for _, record := range []Doc{{"id": "a", "region": "Алматы"}, {"id": "b", "region": ""}, {"id": "c", "region": "Астана"}, {"id": "d", "region": "Алматы"}} {
			if err := tx.Put(ctx, "publication", record); err != nil {
				return err
			}
		}
		return tx.Put(ctx, "article", Doc{"id": "unpublished", "region": "Шымкент"})
	}); err != nil {
		t.Fatal(err)
	}
	regions := array(a.request("GET", "/public/regions", nil, 200, "")["items"])
	if !reflect.DeepEqual(regions, []any{"Алматы", "Астана"}) {
		t.Fatalf("published regions = %v", regions)
	}
}

func TestPublicOpportunityIDFilterStillRequiresApprovedFutureOpportunities(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	if err := a.store.Write(ctx, func(tx *Tx) error {
		for _, record := range []Doc{
			{"id": "active", "status": "approved", "deadline": future},
			{"id": "other-active", "status": "approved", "deadline": future},
			{"id": "expired", "status": "approved", "deadline": past},
			{"id": "draft", "status": "new", "deadline": future},
		} {
			if err := tx.Put(ctx, "opportunity", record); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	filtered := a.request("GET", "/public/opportunities?ids=missing,draft,expired,active", nil, 200, "")
	items := array(filtered["items"])
	if len(items) != 1 || str(object(items[0])["id"]) != "active" || integer(filtered["total"]) != 1 {
		t.Fatalf("filtered opportunities = %v", filtered)
	}
}
