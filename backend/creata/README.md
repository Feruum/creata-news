# Creata API

Independent Go API for Creata News. It uses a fresh PostgreSQL database and does not apply the legacy EMS migrations. Schema v1 is embedded in the binary and applied idempotently at startup. Every database write takes a transaction-scoped advisory lock; job and delivery transitions additionally validate durable leases. This favors workflow correctness for the first release; write throughput is serialized per database.

## Run

Set `DATABASE_URL`, `CREATA_SESSION_SECRET`, `CREATA_WORKER_TOKEN`, and `CREATA_BOT_SERVICE_TOKEN`. All three secrets must contain at least 32 characters and must be different operational credentials. `go run ./cmd/server` starts on port 5555. `-migrate-only` applies schema and exits. `-seed <JSON>` explicitly imports development data and starts the API; it is forbidden with `CREATA_ENV=production`.

`CREATA_ORIGINS` is the exact comma-separated list of browser origins. `CREATA_PUBLIC_URL` builds article links; `CREATA_API_URL` is the externally reachable OAuth callback origin. For deployments using separate frontend/API subdomains, set `CREATA_COOKIE_DOMAIN` to their registrable shared parent, for example `creata.news`; it is applied to both the session and OAuth state cookie. Production OAuth refuses cross-host configuration without this domain, and every configured host must belong to it. Local localhost deployments leave it empty.

Preserve `CREATA_SESSION_SECRET` across restarts: it signs CSRF tokens and encrypts the saved AI credential. Passwords are not stored.

For native local development only, `CREATA_DEV_AUTH=1` exposes `/auth/dev-login` to loopback requests. The development `admin` account also holds chief/editor/author roles so the full workflow can be tested. Production rejects development authentication and private source targets at startup. Docker bridge peers cannot use development login.

## Identity and first administrator

Set `CREATA_GOOGLE_CLIENT_ID` and `CREATA_GOOGLE_CLIENT_SECRET`, or `CREATA_TELEGRAM_CLIENT_ID` and `CREATA_TELEGRAM_CLIENT_SECRET`. Register `<CREATA_API_URL>/api/creata/v1/auth/<provider>/callback` with each provider. The implementation uses OIDC discovery, authorization code + PKCE, single-use browser-bound state, nonce, RSA signature, issuer, audience and expiry verification. Missing providers remain unavailable. Verified identities are keyed by provider subject; matching emails never merge accounts automatically. Return destinations must be relative site paths or exact configured frontend origins.

After a first Google login, run the binary with `-grant-admin-email <verified-email>` to assign staff roles to exactly one verified account. Role assignments are audited. Technical administrator permission alone cannot approve ecosystem articles.

## Storage and integrations

Production requires `CREATA_S3_ENDPOINT`, `CREATA_S3_REGION`, `CREATA_S3_BUCKET`, `CREATA_S3_ACCESS_KEY`, `CREATA_S3_SECRET_KEY`. `CREATA_MEDIA_PUBLIC_URL` optionally points at the bucket/CDN public origin. The API signs S3 PUT requests; bucket read policy/CDN access must be provisioned separately. Development may use `CREATA_MEDIA_DIR` with the local `/media/` handler. Uploads accept only bounded JPEG/PNG/WebP/GIF data and require credit and rights.

Hermes uses `CREATA_WORKER_TOKEN`; Telegram delivery uses the distinct `CREATA_BOT_SERVICE_TOKEN`. Set `CREATA_TELEGRAM_BOT_USERNAME` for account-link URLs and `CREATA_TELEGRAM_CHANNEL_ID` for approved channel editions. The reader bot token and MTProto collection session are owned by their separate runtimes.

AI credentials are write-only through admin settings and AES-GCM encrypted in PostgreSQL. Internal worker configuration requires the worker credential. Endpoint, model, key, positive budget and token prices must be configured before a cost reservation succeeds. Pending reservations count against the daily budget conservatively after a worker crash. Budget accounting uses the PostgreSQL calendar day; digest scheduling uses the configured IANA timezone.

## Delivery outcomes

Scheduled deliveries are leased. A confirmed send is not retried. Expired sending leases and explicitly ambiguous sends become `unknown`; selected content is withheld from subsequent personal editions until staff resolve the delivery. Manual `/digest` sending must acknowledge `/internal/bot/digests/{id}/complete` after all chunks, or `/fail` if transmission fails. Staff inspect `/admin/deliveries` and explicitly `retry` or `mark_sent` unknown outcomes.

## Verification

Run `go test ./...` and `go vet ./...`. PostgreSQL integration tests run when `CREATA_TEST_DATABASE_URL` points at a loopback database whose name contains `test`; each test creates and drops its own random schema. They never truncate the shared development schema. Tests cover publication snapshots, RBAC, CSRF, OIDC signature/claims, parallel AI budget reservation, worker leases, deduplication, the full Hermesâ†’reviewâ†’publicationâ†’digest chain, Telegram linking and delivery uncertainty recovery. Live provider login, real model calls and real Telegram delivery require operator configuration and separate smoke tests.

Editorial staff may submit a URL through `POST /admin/bank/collect` without source administration permission. It queues an inactive, single-page collection source with actor trace. Grouped stories retain source evidence text and immutable raw originals; malformed individual source entries are recorded as warnings while valid entries continue. Approved expired opportunities remain addressable from saved records but do not appear in current feeds or digests.
