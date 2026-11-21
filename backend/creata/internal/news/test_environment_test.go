package news

import (
	"context"
	"testing"
)

func TestVerificationSchemaRejectsUnsafeTargetsBeforeConnecting(t *testing.T) {
	for _, c := range []struct{ env, dsn, name string }{{"production", "postgres://creata@127.0.0.1/creata_test", "verify_0123456789abcdef"}, {"test", "postgres://creata@127.0.0.1/creata", "verify_0123456789abcdef"}, {"test", "postgres://creata@example.org/creata_test", "verify_0123456789abcdef"}, {"test", "postgres://creata@127.0.0.1/creata_test", "public"}, {"test", "postgres://creata@127.0.0.1/creata_test", "verify_abc;DROP SCHEMA public"}} {
		if _, e := IsolatedTestSchema(context.Background(), c.env, c.dsn, c.name, true); e == nil {
			t.Fatal("unsafe verification target accepted", c)
		}
	}
}
