package news

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestSeedLoadsReferencesBeforePublishedArticles(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	for i := 0; i < 24; i++ {
		cid, uid, aid := fmt.Sprintf("seed-category-%d", i), fmt.Sprintf("seed-author-%d", i), fmt.Sprintf("seed-article-%d", i)
		article := validArticle()
		article["id"] = aid
		article["categoryId"] = cid
		article["authorId"] = uid
		article["status"] = "published"
		input, _ := json.Marshal(Doc{"categories": []Doc{{"id": cid, "name": localized("Custom category")}}, "authors": []Doc{{"id": uid, "name": "Seed author"}}, "articles": []Doc{article}})
		if err := a.store.Seed(ctx, input); err != nil {
			t.Fatalf("seed order failed at %d: %v", i, err)
		}
		published, err := a.store.Get(ctx, "publication", aid)
		if err != nil || str(published["categoryId"]) != cid || str(published["authorId"]) != uid {
			t.Fatal("seed publication lost references")
		}
	}
}
