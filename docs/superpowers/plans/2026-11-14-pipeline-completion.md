# Creata News pipeline completion

> Execution: subagent-driven-development, existing UI preserved. The user approved implementing the audit in this session; no new design approval is required. This workspace has no Git repository, so changes stay in the shared workspace and no commits/worktrees are created.

**Goal:** Close every code gap in PIPELINE-AUDIT-2026-11-14.md and make backend the source of all editorial content. External credentials remain configurable; the AI provider is deliberately deferred by the user.

**Architecture:** Keep Next/React/Vite, Go/PostgreSQL, Python Hermes and the existing Creata components. Extend the current versioned-document store and API. AI checkpoints belong to leased jobs; public content always comes from approved server snapshots. Import original business content and assets once into backend storage, preserving rendering and URLs.

## Work and acceptance checklist

- [x] Server integrity: incomplete opportunities remain drafts; approval validates three languages, organization/type/country, links/deadlines and photo rights. Article author/category references must exist at write and publication. Add negative API tests.
- [x] Hermes checkpoints: persist research, draft, review under the job lease, display progress, resume successful steps after retry, preserve editorial expectedRevision protection. Add crash/retry and stale-lease tests.
- [x] Research schema and UI: store claims with source URLs, numbers/dates, conflicts and one-source flags; render the saved brief in the existing article editor.
- [x] Related opportunities: server selects approved, live, relevant opportunities, passes them to all draft/rework jobs, validates returned IDs and persists article associations; render with existing opportunity components.
- [x] Story similarity: conservative explained candidates and reversible editor merge/split retaining raw sources; no silent destructive merge. Test repeat ingestion and merge undo.
- [x] Backend CMS: versioned localized ecosystem/about documents with preview, chief/editor approval, conflict checks and audit. Existing public components receive backend data and existing admin primitives edit it.
- [x] Original newspaper: import issue 01 HTML and catalog into backend publication storage; remove frontend fallback, protect the original reserved slug from conflicting publications and preserve the original renderer, images, layout and routes.
- [x] Media ownership: import business assets into API media storage, rewrite content URLs, list/search/update rights/delete unreferenced media, consistent image types, compensate failed persistence after upload and expose cleanup. Keep UI fonts/icons as application assets.
- [x] Analytics: calculate editorial stage counts and durations from audit history, expose in the existing analytics screen.
- [x] Reproducible run: one configured port map for API/web/admin/Hermes/bot, deterministic launcher, explicit occupied-port diagnostics, no stopping unrelated projects.
- [x] CI: isolated PostgreSQL, API/Python integration, public/admin browser scenarios, build/types/contracts checks; no paid AI or real Telegram needed.
- [x] Verification: run API, Python, Bun and browser tests, clean production builds, migration idempotence, persisted-content restart checks, and full controlled source→draft→editor→public→digest/delivery ledger scenario. Record external credential prerequisites accurately.

## Ownership

Pipeline implementer owns jobs/editorial/domain/opportunity backend files, Python Hermes pipeline/client, existing article/bank/Hermes workflow screens and their tests. Root owns CMS/media/newspaper migration, shared API contracts, CMS/admin routing, local scripts/CI and final integration. Shared file changes are coordinated before edits.

## Verification commands

`go test -count=1 ./...` in backend/creata against isolated PostgreSQL; `python -m unittest discover -s hermes-worker/tests -v`; `bun test packages creatanews/src events-admin/src/creata agent-admin/src/creata telegram-bot/src/creata`; `bun run typecheck`; `bun scripts/generate-contracts.ts --check`; `bun run build`; public/editorial Playwright suites; `bun scripts/test-live-pipeline.ts` against isolated API. Production screenshots compare existing layouts at 360/768/1440 in both themes.
