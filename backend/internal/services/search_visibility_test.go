//go:build integration

package services

import (
	"connectrpc.com/connect"
	"context"
	"github.com/meilisearch/meilisearch-go"
	searchv1 "github.com/studyverse/ems-backend/gen/search/v1"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"testing"
)

type staleSearchBackend struct{}

func (staleSearchBackend) GlobalSearch(context.Context, string, int32) (*search.MultiSearchResult, error) {
	return &search.MultiSearchResult{TotalHits: 5, Results: []search.SearchResult{
		{Type: search.IndexEvents, ID: 1, Title: "old public title"},
		{Type: search.IndexEvents, ID: 2, Title: "private draft"},
		{Type: search.IndexEvents, ID: 3, Title: "archived"},
		{Type: search.IndexEvents, ID: 4, Title: "deleted"},
		{Type: search.IndexEvents, ID: 5, Title: "Sunday only"},
		{Type: search.IndexUsers, ID: 1, Title: "alice", Description: "private@example.com"},
	}}, nil
}
func (staleSearchBackend) SearchEvents(context.Context, string, int32, string) (*meilisearch.SearchResponse, error) {
	return nil, nil
}

func TestGlobalSearchRejectsStalePrivateResultsAndUserEmails(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE events (LIKE public.events INCLUDING ALL);
	INSERT INTO events (id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
	VALUES (1,'current public title','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','a','a'),
	(2,'private draft','{}',1,1,NOW(),NOW()+interval '1 hour','{}','draft','b','b'),
	(3,'archived','{}',1,1,NOW(),NOW()+interval '1 hour','{}','archived','c','c'),
	(5,'Sunday only','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','d','d');
	UPDATE events SET publish_on_map=false,publish_on_sunday_open=true WHERE id=5;`); err != nil {
		t.Fatal(err)
	}
	svc := &SearchService{db: db, searchClient: staleSearchBackend{}}
	response, err := svc.GlobalSearch(ctx, connect.NewRequest(&searchv1.GlobalSearchRequest{Query: "secret"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.Results) != 1 || response.Msg.Results[0].Id != 1 || response.Msg.Results[0].Title != "current public title" || response.Msg.TotalHits != 1 {
		t.Fatalf("stale/private search data escaped: %v", response.Msg)
	}
}
