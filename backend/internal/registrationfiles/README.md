# Private registration attachments

`registrationfiles.New(db, store)` accepts the independently constructed
`storage.NewPrivateProvider(config)`. If construction fails, pass `nil` and keep
the feature unavailable: `Available()` is used by form publishing to reject FILE
questions. `Start(ctx)` runs expiry cleanup until the application stops.

The server-only `STORAGE_PRIVATE_CONTAINER` defaults to `ems-registration-files`.
It must differ from the public image container. The adapter reuses the configured
provider, endpoint, account credentials and region. S3-compatible provisioning is
in `infra/storage/init-private.sh`, called by development and VPS compose. It
creates a separate bucket, removes its bucket policy, and configures exact client
origins. Never give this bucket the anonymous image bucket policy. For Azure,
create a container with public access disabled; the adapter verifies this before
enabling file questions. Azure CORS is account-wide, so existing rules must only
name configured client origins; wildcard or unrelated origins disable private
files rather than changing existing public image rules.

Staging PUT links expire in five minutes. Completion reads at most the declared
size plus one byte, validates the extension, MIME and file contents, then writes
the accepted bytes under a different final key using an exclusive create. Only
staging keys can be signed for PUT. Final files carry `private, no-store` cache
control and attachment content disposition. Read links expire in one minute and
require the owner or an authenticated event registration manager.

Files are PDF, JPEG, PNG, WebP, DOCX or XLSX, at most 10 MiB per file and three
files per question; field rules may narrow those limits. Office documents must
have the matching OOXML package structure. Unknown owners, stale forms, closed
or source-hidden events, foreign files and already-bound files are rejected.
`ValidateAndAttach` runs in the registration transaction, verifies response,
event, version, field and owner identity, and binds only ready files. Rollback
also rolls back file binding.

Upload reservations expire after 24 hours. The cleanup worker retries failed
object deletions and never deletes attached final objects. It also removes
attached files' obsolete staging objects after their signed write window has
elapsed. Owner reservations and recent upload requests are capped to prevent
unbounded staging growth. The quota is twice the published form's total file
capacity, at least 30 and at most 300. It allows replacing a ready file or
retrying a failed upload while keeping the selected-answer `maxFiles` limit.

For a VPS, `STORAGE_PUBLIC_URL` supplies the external storage host and proxy
prefix. The S3 adapter signs the external host and the canonical `/bucket/key`
path, then prepends `/storage` to the outward URL. Caddy strips that prefix and
preserves Host before proxying to RustFS. Only private staging PUT/OPTIONS are
allowed through this proxy, with a 10 MiB request body cap; public image writes
remain unavailable there. Changing this rewrite or Host handling requires
changing the signer and its proxy-path test together.

Run focused unit tests with `go test ./internal/registrationfiles ./internal/storage`.
To run the lifecycle test, set `REGISTRATION_FILES_TEST_DSN` to a disposable
PostgreSQL database. Tests create and drop a randomly named schema without
accessing existing application data.
