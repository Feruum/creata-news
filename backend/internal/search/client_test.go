package search

import (
	"context"
	"fmt"
	"github.com/meilisearch/meilisearch-go"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchMutationsRetryFailedOrCanceledAsynchronousTasks(t *testing.T) {
	for _, status := range []string{"failed", "canceled", "succeeded"} {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/tasks/1" {
					fmt.Fprintf(w, `{"uid":1,"status":%q,"error":{"message":"disk full","code":"database_size_limit_reached"}}`, status)
					return
				}
				w.WriteHeader(http.StatusAccepted)
				fmt.Fprint(w, `{"taskUid":1}`)
			}))
			defer server.Close()
			client := &Client{meili: meilisearch.New(server.URL)}
			ctx := context.Background()
			for name, operation := range map[string]func() error{
				"event":         func() error { return client.IndexEvent(ctx, &EventDocument{ID: 1}) },
				"events":        func() error { return client.IndexEvents(ctx, []EventDocument{{ID: 1}}) },
				"organization":  func() error { return client.IndexOrganization(ctx, &OrganizationDocument{ID: 1}) },
				"organizations": func() error { return client.IndexOrganizations(ctx, []OrganizationDocument{{ID: 1}}) },
				"user":          func() error { return client.IndexUser(ctx, &UserDocument{ID: 1}) },
				"users":         func() error { return client.IndexUsers(ctx, []UserDocument{{ID: 1}}) },
				"delete":        func() error { return client.DeleteDocument(ctx, IndexEvents, 1) },
				"delete batch":  func() error { return client.DeleteDocuments(ctx, IndexEvents, []int32{1, 2}) },
			} {
				if err := operation(); (err == nil) != (status == "succeeded") {
					t.Errorf("%s task %s acknowledged incorrectly: %v", name, status, err)
				}
			}
		})
	}
}
