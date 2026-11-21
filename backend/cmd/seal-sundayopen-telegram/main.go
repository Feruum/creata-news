// seal-sundayopen-telegram reads a private credential JSON object from stdin and
// emits only a deployment-scoped sealed bundle suitable for a GitOps overlay.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/studyverse/ems-backend/internal/telegramconfig"
)

var errInput = errors.New("could not seal Sunday Open Telegram configuration")

func sealInput(input io.Reader, output io.Writer, publicKey, keyID string) error {
	raw, err := io.ReadAll(io.LimitReader(input, 4097))
	if err != nil || len(raw) > 4096 {
		return errInput
	}
	defer clear(raw)
	var credentials telegramconfig.Credentials
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&credentials); err != nil {
		return errInput
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errInput
	}
	sealed, err := telegramconfig.Seal(telegramconfig.Descriptor{
		Algorithm: telegramconfig.Algorithm, PublicKey: publicKey, KeyID: keyID,
	}, credentials)
	if err != nil {
		return errInput
	}
	if _, err := fmt.Fprintln(output, sealed); err != nil {
		return errInput
	}
	return nil
}

func main() {
	publicKey := flag.String("public-key", "", "publicKey from the authenticated HTTPS setup-key response")
	keyID := flag.String("key-id", "", "keyId from that same response")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Credentials must be supplied through stdin")
		os.Exit(1)
	}
	if err := sealInput(os.Stdin, os.Stdout, *publicKey, *keyID); err != nil {
		fmt.Fprintln(os.Stderr, errInput)
		os.Exit(1)
	}
}
