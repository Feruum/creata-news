package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/studyverse/ems-backend/internal/telegramconfig"
)

func TestOperatorHelperReadsCredentialsOnlyFromStdin(t *testing.T) {
	secret, scope := strings.Repeat("s", 64), "https://map.webspace.cat"
	key, _ := telegramconfig.PublicKey(secret, scope)
	want := telegramconfig.Credentials{
		BotToken: "12345678:" + strings.Repeat("a", 35), BotUsername: "sundayopen_bot",
		WebhookSecret: strings.Repeat("w", 43),
	}
	input, _ := json.Marshal(want)
	var output bytes.Buffer
	if err := sealInput(bytes.NewReader(input), &output, key.PublicKey, key.KeyID); err != nil {
		t.Fatal(err)
	}
	sealed := strings.TrimSpace(output.String())
	if strings.Contains(sealed, want.BotToken) || strings.Contains(sealed, want.WebhookSecret) {
		t.Fatal("operator output exposed plaintext credentials")
	}
	got, err := telegramconfig.Open(secret, scope, sealed)
	if err != nil || got != want {
		t.Fatal("operator helper produced an unreadable credential bundle")
	}
	for _, invalid := range []string{"not-json", `{}`, string(input) + `{}`, strings.Repeat(" ", 4097)} {
		output.Reset()
		if err := sealInput(strings.NewReader(invalid), &output, key.PublicKey, key.KeyID); err == nil || output.Len() != 0 {
			t.Fatal("helper accepted invalid input or printed partial credentials")
		}
	}
}
