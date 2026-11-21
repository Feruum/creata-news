//go:build integration

package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestReindexRemovesLegacyPrivateAndDeletedEvents(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE events (LIKE public.events INCLUDING ALL);
	CREATE TABLE organizations (LIKE public.organizations INCLUDING ALL);
	INSERT INTO organizations (id,title,organization_type_id) VALUES (1,'org',1);
	INSERT INTO events (id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
	VALUES (1,'draft','{}',1,1,NOW(),NOW()+interval '1 hour','{}','draft','a','a'),
	(2,'archived','{}',1,1,NOW(),NOW()+interval '1 hour','{}','archived','b','b'),
	(4,'current published','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','c','c'),
	(5,'Sunday only','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','d','d');
	UPDATE events SET publish_on_map=false,publish_on_sunday_open=true WHERE id=5;`); err != nil {
		t.Fatal(err)
	}
	documents := map[int32]EventDocument{1: {ID: 1}, 2: {ID: 2}, 3: {ID: 3}, 5: {ID: 5}}
	failTasks := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/tasks/1":
			status := "succeeded"
			if failTasks {
				status = "failed"
			}
			fmt.Fprintf(w, `{"uid":1,"status":%q,"error":{"code":"database_size_limit_reached"}}`, status)
		case r.URL.Path == "/indexes/events/documents/fetch":
			items := []EventDocument{}
			for _, doc := range documents {
				items = append(items, doc)
			}
			json.NewEncoder(w).Encode(map[string]any{"results": items, "offset": 0, "limit": 1000, "total": len(items)})
		case r.URL.Path == "/indexes/events/documents/delete-batch":
			var ids []string
			if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
				t.Error(err)
			}
			if !failTasks {
				for _, id := range ids {
					value, err := strconv.Atoi(id)
					if err != nil {
						t.Error(err)
					}
					delete(documents, int32(value))
				}
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"taskUid":1}`)
		case r.Method == http.MethodDelete:
			id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/indexes/events/documents/"))
			if err != nil {
				t.Error(err)
			}
			if !failTasks {
				delete(documents, int32(id))
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"taskUid":1}`)
		case r.Method == http.MethodPost:
			var items []EventDocument
			if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
				t.Error(err)
			}
			if !failTasks {
				for _, doc := range items {
					documents[doc.ID] = doc
				}
			}
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"taskUid":1}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	indexer := NewIndexer(&Client{meili: meilisearch.New(server.URL)}, db)
	count, err := indexer.ReindexEvents(ctx)
	if err != nil || count != 1 || len(documents) != 1 || documents[4].Title != "current published" {
		t.Fatalf("stale index cleanup: count=%d docs=%+v error=%v", count, documents, err)
	}
	failTasks = true
	if _, err := indexer.ReindexEvents(ctx); err == nil {
		t.Fatal("reindex acknowledged a failed task")
	}
}
