package news

import (
	"context"
	"encoding/json"
	"errors"
)

func (s *Store) GrantAdmin(ctx context.Context, email string) error {
	return s.Write(ctx, func(t *Tx) error {
		rows, e := t.Query(ctx, "SELECT data FROM creata_records WHERE kind='user' AND lower(data->>'email')=lower($1) FOR UPDATE", email)
		if e != nil {
			return e
		}
		matches := []Doc{}
		for rows.Next() {
			var b []byte
			if e = rows.Scan(&b); e != nil {
				rows.Close()
				return e
			}
			var u Doc
			if e = json.Unmarshal(b, &u); e != nil {
				rows.Close()
				return e
			}
			matches = append(matches, u)
		}
		rows.Close()
		if len(matches) != 1 {
			return errors.New("exactly one verified-email account must match")
		}
		u := matches[0]
		u["roles"] = []any{"reader", "author", "editor", "chief_editor", "admin"}
		if e = t.Put(ctx, "user", u); e != nil {
			return e
		}
		if e = t.Put(ctx, "author", Doc{"id": u["id"], "name": u["name"], "bio": localized("Creata News editorial team"), "role": "editor"}); e != nil {
			return e
		}
		return t.Audit(ctx, Doc{"id": "operator"}, "user.bootstrap", str(u["id"]), nil)
	})
}
