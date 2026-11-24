# CR≡ATA News

Independent RU / ҚАЗ / EN editorial news platform. Hermes collects and researches stories. Editors approve a specific revision for the website, select social accounts for Postiz, and manage Telegram digests. The [Russian pipeline guide](docs/PIPELINE.md) covers the workflow, local startup, delivery states and verification.

Visual implementation preserves the original public frontend and admin component owners. All three apps share `@creata/brand`: Creata violet, neutral surfaces, the existing local Manrope/Kazakh fonts and one logo. The public app uses the original Spotlight/poster rail and a single compact navigation; admin workflows use the original Sidebar and UI primitives. Rest of World and Skiper are supporting references.

## Applications

| Application | Address | Responsibility |
|---|---|---|
| Public Next.js site | http://localhost:3002/ru | News, categories, authors, opportunities, search, accounts and saved items |
| Editorial React app | http://localhost:6868 | Bank, kanban, article revisions, publication, digests and analytics |
| Hermes React app | http://localhost:5174 | Sources, jobs, selection rules, AI configuration and failures |
| Go API | http://localhost:5555/health | New PostgreSQL schema, authentication, permissions and durable workflows |
| Python Hermes worker | Background service | RSS/HTML/Telegram collection and AI tasks |
| Bun/Grammy bot | http://localhost:3031/health/live | Preferences and authorized Telegram deliveries |

## Native local development

Requirements: Bun 1.4.2, Go (version in backend/creata/go.mod), Python with venv (3.12 verified locally, 3.13 in CI), Docker with Compose for the standard project database.

The complete native stack starts with `bun run dev` (or `bun run dev:all`). The launcher checks the database connection and starts the existing project PostgreSQL container when its local port is down.
It starts API, public site, both admins, Hermes and the delivery bot; chooses free
IPv4/IPv6 ports; and writes the actual URLs to `.runtime/local-ports.json`.
Set `CREATA_PYTHON` when Python is installed outside PATH. Optional fixed ports
are `CREATA_API_PORT`, `CREATA_WEB_PORT`, `CREATA_EDITORIAL_PORT`,
`CREATA_HERMES_PORT`, `CREATA_BOT_PORT`. An occupied fixed port produces a clear
error and leaves other projects running. The default bot port is now **3031**.

The launcher checks Python and the pinned Telethon dependency. It can create or repair the project environment in `.runtime/hermes-venv`. An explicitly configured external interpreter needs the dependencies in `hermes-worker/requirements.txt`. Bot delivery uses Bun and its own Telegram bot token.

Original ecosystem/about/newspaper content and business assets are imported
idempotently from `backend/creata/content` into PostgreSQL and API/S3 storage.
Edit published pages in **Контент сайта** and manage assets and rights in
**Медиатека**. Old media URLs resolve through the API. Local `uploads/` holds
runtime data and is ignored by Git. Keep it with the local PostgreSQL backup.

Hermes uses `CREATA_WORKER_STATE_DIR` for its durable paid-response outbox during
API interruptions; Compose mounts a separate `hermes-state` volume. Keep this
directory persistent. AI endpoint/model/key and budget can be configured later.

For the Compose setup, `bun run setup:local --test-db` provisions the sibling `creata_test` database and records its address in the private environment. `bun run test:pipeline` starts its own API, builds and serves the public site, and exercises Hermes, editorial approval, Postiz, Telegram and both administrative apps using local HTTP fixtures. CI runs the same command.

```sh
bun install --frozen-lockfile
bun run setup:local
docker compose up -d postgres
bun run dev
```

`bun run dev:api` remains available for backend-only work. `bun run dev:ui`
starts only the three frontend packages when an API is already managed separately.

The setup command creates a private `.env` with independent random service secrets and preserves an existing file. It enables **local loopback-only development sign-in** for testing roles. Production rejects this mode. New article banks start empty; original supplied company pages and newspaper №01 migrate as existing approved content. Demo news and statistics are never created automatically.

To use the supplied reference content explicitly in development:

```sh
bun run seed:dev
# Stop a running API first, then start it with the seed:
bun scripts/run-api.ts -seed /absolute/path/to/.runtime/dev-seed.json
```

The seed imports eight reference articles, 19 categories, 10 authors and six demo opportunities with the original prototype photos and localized audiences. Five opportunities are approved, including the Creata Games hackathon; the “Mama Pro” hackathon retains its pending review status and stays private. Reference media is development material; confirm reuse rights before publishing. The production executable refuses development seeding. The original apps are backed up in `archives/pre-creata-2026-11-14.zip`, with an internal per-file manifest and an external archive SHA256 report. The original prototype and PDF remain untouched.

## Docker deployment

```sh
bun run setup:local
docker compose up --build -d
```

Compose runs an isolated Creata database, API, three frontends, Hermes worker and digest bot. PostgreSQL and frontend ports bind to loopback. Container development sign-in is disabled because requests arrive through a proxy; use native local development for role testing, or configure OAuth for container use.

For production, set `CREATA_ENV=production`, `CREATA_DEV_AUTH=0`, strong service/session secrets, exact HTTPS frontend origins and public URLs, OAuth clients, and S3 credentials. The server refuses production startup with dev authentication, private-source allowances, HTTP public URLs or missing S3 configuration. Use your HTTPS ingress in front of the loopback-bound services. No source deployment modifies existing AITU databases.

For separate production hosts (for example `news.example.com`, `editorial.example.com`, `hermes.example.com`, `api.example.com`), configure `CREATA_COOKIE_DOMAIN=example.com` and matching exact `CREATA_ORIGINS`/`CREATA_API_URL`. All hosts must belong to that cookie domain. Both OAuth state and account cookies use it; cross-host OAuth without a matching domain is rejected. A deployment on one hostname needs no cookie domain. `CREATA_EDITORIAL_URL` and `CREATA_HERMES_ADMIN_URL` set cross-app links when building admin containers.

### Authentication and staff

Configure `CREATA_GOOGLE_CLIENT_ID`, `CREATA_GOOGLE_CLIENT_SECRET`, `CREATA_API_URL`, and Google callback `/api/creata/v1/auth/google/callback`. Telegram website login uses `CREATA_TELEGRAM_CLIENT_ID` and `CREATA_TELEGRAM_CLIENT_SECRET`, callback `/api/creata/v1/auth/telegram/callback`. Unconfigured providers are unavailable in the UI.

After a verified-email account has signed in, bootstrap staff explicitly:

```sh
# From backend/creata with its environment configured:
go run ./cmd/server -grant-admin-email editor@example.com
```

Roles and vertical assignments are managed in the editorial team/settings screen. Authors edit their own drafts; editors publish assigned world verticals; chief editors approve ecosystem articles. Hermes can submit drafts but cannot publish. Google/Telegram subjects are bound to accounts; contact emails alone never merge identities.

### Hermes

Install its one external integration dependency and start the worker with the same API environment:

```sh
python -m pip install -r hermes-worker/requirements.txt
# Set CREATA_API_ORIGIN and CREATA_WORKER_TOKEN from the private environment.
cd hermes-worker
python -m hermes.main
```

Add sources in the Hermes app. Set an OpenAI-compatible endpoint, model, API key, token prices and a positive daily budget in AI settings. Keys are write-only and encrypted at rest. With no AI configuration, source collection and manual editing work; AI jobs show `not_configured`. Each model request reserves budget before transmission and records actual usage afterward. Unknown provider outcomes conservatively consume the reservation.

Editors can submit an individual URL from the news bank without managing the full source registry. These one-off jobs fetch only the submitted page. Research and review use retrieved evidence texts, and a reference URL without readable evidence never counts as a corroborating source. Provider redirects are refused so authorization headers cannot be forwarded to another origin.

RSS and website targets are resolved and pinned to public IPs, bounded in size, and checked again across redirects. Crawlers respect robots.txt. JavaScript-only/paywalled pages report collection failures; no content is invented. The environment flag `CREATA_ALLOW_PRIVATE_SOURCES=1` is restricted to controlled development fixtures and rejected in production.

Telegram source collection is separate from the digest bot. Configure `TELEGRAM_API_ID` and `TELEGRAM_API_HASH`, then authorize a dedicated collector interactively:

```sh
cd hermes-worker
python -m hermes.telegram_login
```

The helper saves a private session file without printing its contents. Set `TELEGRAM_COLLECTOR_SESSION_FILE` to that path, or provide `TELEGRAM_COLLECTOR_SESSION` through your secret store. The collector reads only configured public channel URLs.

### Telegram delivery

Configure `TELEGRAM_BOT_TOKEN`, `CREATA_BOT_SERVICE_TOKEN`, `CREATA_TELEGRAM_BOT_USERNAME`, and `CREATA_TELEGRAM_CHANNEL_ID`. Make the bot an administrator of the intended channel. Native development can use polling. Production requires `TELEGRAM_WEBHOOK_URL` and a 32-character-or-longer `TELEGRAM_WEBHOOK_SECRET`.

```sh
bun run --filter @creata/telegram-bot start
```

Commands: `/start`, `/digest`, `/settings`, `/pause`, `/resume`, `/help`. Personal digests use the selected language/topics/frequency. Channel editions require editorial approval. Confirmed deliveries are never retried automatically; unknown outcomes require an editor to inspect Telegram and resolve the delivery ledger.

Default schedule: 09:00 daily or Friday 09:00, `Asia/Qyzylorda`. Channel language defaults to Russian. Short content inventories produce shorter digests.

### Postiz social publishing

Configure the API server with `CREATA_POSTIZ_API_KEY` and the full Postiz Public API base in `CREATA_POSTIZ_API_URL`. The default is `https://api.postiz.com/public/v1`. In the article or chart editor, approve the revision, choose **Publish** or **Schedule**, then select accounts and languages. The default `CREATA_POSTIZ_MODE=draft` creates a draft on each selected account. Set it to `now` to enable platform delivery after the site publication becomes effective. Account-specific delivery states appear on the material. [Setup, media prerequisites and retry behavior](docs/PIPELINE.md)

## Verification

```sh
bun run contracts
bun scripts/generate-contracts.ts --check
bun run typecheck
bun run build
bun run test
cd backend/creata
go test ./...
```

Set `CREATA_TEST_DATABASE_URL` to a disposable PostgreSQL database to enable backend integration tests; test schemas are isolated. Python checks:

```sh
python -m unittest discover -s hermes-worker/tests -v
```

For browser smoke tests, run the seeded API and three apps with explicit native development auth, install Chromium with `node node_modules/@playwright/test/cli.js install chromium --only-shell`, then `bun run test:e2e`. Screenshots and results are written under ignored `.runtime/`. Set `PLAYWRIGHT_BROWSERS_PATH` when using a custom browser cache.

Live Google/Telegram authentication, paid AI and actual Telegram sends require operator credentials and are separate from local fixture verification.

## Data and operations

New schema migrations run before the API starts. Articles retain immutable revisions and separate published snapshots; saving an edited draft does not replace the public version. Approval is tied to one revision, and stale saves return 409. Source jobs and deliveries use durable leases. Readiness checks query PostgreSQL.

Back up the Creata PostgreSQL database with `pg_dump`, the S3 bucket (or development media volume), and service/session secrets. Restore into a separate database first and check published snapshots, drafts and the delivery ledger before activating schedulers. Preserve the session secret to retain sessions and decrypt stored integration keys. Do not run schedulers against a restored test copy configured with live Telegram credentials.

The API contract is in `docs/API-CONTRACT.md` and `backend/creata/openapi.json`. Frontends consume generated schemas from `packages/contracts`. The approved implementation boundaries and release gates are in `docs/IMPLEMENTATION.md`.
