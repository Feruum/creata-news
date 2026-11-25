package news

import (
	"context"
	"testing"
)

func TestCorrectSeededDesignathonTitle(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		if err := tx.Put(ctx, "opportunity", Doc{"id": "o-designthon", "title": Doc{"ru": "Designthon: дизайн-марафон TransforNATION", "en": "Designthon: TransforNATION design marathon", "kk": "Редакционная правка"}}); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, schema+publicCatalogSchema)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	opportunity, err := a.store.Get(ctx, "opportunity", "o-designthon")
	if err != nil {
		t.Fatal(err)
	}
	title := object(opportunity["title"])
	if str(title["ru"]) != "Designathon: дизайн-марафон TransforNATION" || str(title["en"]) != "Designathon: TransforNATION design marathon" || str(title["kk"]) != "Редакционная правка" {
		t.Fatal(title)
	}
}
