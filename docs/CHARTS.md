# Creata Data & Charts

Public gallery: `/ru/charts`, `/kk/charts`, `/en/charts`; detail: `/{locale}/charts/{slug}`. The existing disclosure menu links to the gallery. `/charts` redirects to Russian. The section uses the current Creata page/panel colors in both themes; it does not add another primary navigation.

Editorial: `/charts` lists graphics; `/charts/{articleId}` edits them. Hermes control: `/charts` selects a collected story and queues a graphic draft. Editorial articles have optional `format: "graphic"` and `graphic: GraphicData`, preserving the existing revisions, role checks, 409 conflicts, source references, approval, scheduling, withdrawal and published snapshots. No separate publishing permission is introduced.

## Creating a graphic

1. Create a chart and choose a vertical and data source. Enter RU/ҚАЗ/EN explanation and methodology.
2. Select bars, lines, area or donut; specify a unit and period. Enter row labels and values or import CSV. CSV has a label column followed by 1–6 numeric columns, uses a decimal point and accepts quoted commas/newlines. Missing cells are rejected, never interpreted as zero. A dataset contains 2–120 rows.
3. Inspect the live preview, verify the figures against the original source, and tick data verification. Editing the dataset clears that checkbox.
4. Save, approve the exact revision and publish now or schedule. Any later edit revokes approval; the public page keeps the previous published snapshot until the new revision is approved and published.

Readers can inspect points with the keyboard, expand an accessible table and download CSV. Negative numeric values are preserved. Bar axes include zero; donut values must be non-negative with a positive total. Arbitrary HTML, scripts and third-party chart embeds are not accepted.

## Hermes

From the news bank in Hermes's Charts section, create a graphic from an existing source story. Alternatively, add a source URL when creating a manual graphic and select “Draft with Hermes”. The regular durable `draft`/`rework` jobs carry `article.format=graphic`; worker leases, budgets and retries remain shared.

Every proposed value must have a literal number and quote from retrieved source text. The worker rejects invented values, absent evidence cells, unretreived URLs, incompatible series and empty/negative donut totals. It does not calculate new values, convert currencies, interpolate data or infer missing totals. It prepares three language versions and review notes, always setting `verified=false`. Insufficient comparable source figures fail explicitly. Real AI requests require a configured provider, token prices and budget; disabled providers produce the existing “Not configured” state.

API additions: optional `Article.format`/`Article.graphic`, `format=graphic` filter on public/admin article lists, `POST /api/creata/v1/admin/articles/{id}/graphic-draft` with `expectedRevision`, and `POST /api/creata/v1/admin/bank/{id}/graphic`. The OpenAPI contract and generated TypeScript schemas include these additions. Story/queue retries reuse the same candidate and pending job.

## Checks

Dataset validation and CSV tests: `bun test packages/graphics`. API integration tests: `go test ./...` in `backend/creata` with the disposable `CREATA_TEST_DATABASE_URL`. Worker numeric-evidence tests are included in Python unittest discovery. Browser regression: `bunx playwright test --config playwright.creata.config.ts graphics.spec.ts --output ../.runtime/charts-e2e-results` in `events-admin` with explicitly enabled local development sign-in; it creates only named QA data, publishes briefly for verification, then withdraws/rejects its own record.
