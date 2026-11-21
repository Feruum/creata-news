//go:build integration

package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestReindexBatchesLegacyCleanupAndCurrentPublishedEvents(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE events (LIKE public.events INCLUDING ALL);
 CREATE TABLE organizations (LIKE public.organizations INCLUDING ALL);
 INSERT INTO organizations(id,title,organization_type_id) VALUES(1,'current org',1);
 INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
 SELECT n,'current published '||n,'{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','batch-'||n,'batch-'||n FROM generate_series(1,201) n;
 INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
 VALUES(202,'private draft','{}',1,1,NOW(),NOW()+interval '1 hour','{}','draft','private202','private202'),(203,'private archived','{}',1,1,NOW(),NOW()+interval '1 hour','{}','archived','private203','private203');`); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	documents := make(map[int32]EventDocument, 1001)
	for id := int32(1); id <= 1001; id++ {
		documents[id] = EventDocument{ID: id, Title: "legacy"}
	}
	addTasks, deleteTasks, fetchPages := 0, 0, 0
	maxBatch := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/tasks/1" {
			fmt.Fprint(w, `{"uid":1,"status":"succeeded"}`)
			return
		}
		if r.URL.Path == "/indexes/events/documents/fetch" {
			var request struct{ Offset, Limit int }
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			ids := make([]int, 0, len(documents))
			for id := range documents {
				ids = append(ids, int(id))
			}
			sort.Ints(ids)
			end := min(request.Offset+request.Limit, len(ids))
			items := make([]EventDocument, 0, end-request.Offset)
			for _, id := range ids[request.Offset:end] {
				items = append(items, documents[int32(id)])
			}
			fetchPages++
			json.NewEncoder(w).Encode(map[string]any{"results": items, "offset": request.Offset, "limit": request.Limit, "total": len(ids)})
			return
		}
		if addTasks+deleteTasks >= 22 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"message":"per-ID reindex exceeds bounded task count","code":"too_many_tasks"}`)
			return
		}
		switch {
		case r.URL.Path == "/indexes/events/documents/delete-batch":
			var ids []string
			if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			maxBatch = max(maxBatch, len(ids))
			deleteTasks++
			for _, id := range ids {
				n, err := strconv.ParseInt(id, 10, 32)
				if err != nil {
					t.Error(err)
				}
				delete(documents, int32(n))
			}
		case r.Method == http.MethodDelete:
			id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/indexes/events/documents/"), 10, 32)
			if err != nil {
				t.Error(err)
			}
			delete(documents, int32(id))
			deleteTasks++
		case r.Method == http.MethodPost && r.URL.Path == "/indexes/events/documents":
			var items []EventDocument
			if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			maxBatch = max(maxBatch, len(items))
			addTasks++
			for _, item := range items {
				documents[item.ID] = item
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"taskUid":1}`)
	}))
	defer server.Close()
	count, err := NewIndexer(&Client{meili: meilisearch.New(server.URL)}, db).ReindexEvents(ctx)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		t.Fatalf("reindex exceeded bounded batch tasks: add=%d delete=%d error=%v", addTasks, deleteTasks, err)
	}
	if count != 201 || len(documents) != 201 {
		t.Fatalf("legacy/private cleanup incomplete: indexed=%d remaining=%d", count, len(documents))
	}
	for id := int32(1); id <= 201; id++ {
		doc := documents[id]
		if doc.Title != fmt.Sprintf("current published %d", id) || doc.OrganizationTitle != "current org" {
			t.Fatalf("event %d did not use current SQL document: %+v", id, doc)
		}
	}
	if addTasks != 3 || deleteTasks != 9 || fetchPages != 2 || maxBatch > 100 {
		t.Fatalf("unbounded requests: add=%d delete=%d fetch=%d max=%d", addTasks, deleteTasks, fetchPages, maxBatch)
	}
}
