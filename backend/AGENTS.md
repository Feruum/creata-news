# Creata API

Work from `backend/creata`. The Go service starts in `cmd/server/main.go`, and `internal/news` contains its HTTP handlers, storage, and schema.

- Run `go test ./...` and `go vet ./...` from `backend/creata`.
- Set `CREATA_TEST_DATABASE_URL` to a disposable local PostgreSQL database to run integration tests.
- Update `backend/creata/openapi.json` for contract changes and run `bun run contracts` from the repository root.

See [API setup](creata/README.md) for storage, identity, and integration settings.
