# Creata interactive demo

## Current homepage release

Published 16 November 2026 (Asia/Qyzylorda) to the existing `ferumms-projects/creata-news-demo` project. Production deployment: `dpl_CdbF7k85DoS2GAeMnDeBr2Ru9g8X`; Vercel reports Ready and the public alias points to this release.

- Classic homepage: https://creata-news-demo.vercel.app/ru
- Editorial V2 homepage: https://creata-news-demo.vercel.app/ru/v2
- Both variants also support `/kk` and `/en`; switching locale preserves `/v2`.

This release includes the generated human portrait in V2, topic-specific homepage story covers, the shared reference-shaped wordmark with the brand blue NEWS accent, and the flat dark newspaper promotion. The deployment allowlist includes only the two used generated homepage images and the article-cover folder.

The isolated frontend package passed its production build with demo mode enabled and an unreachable backend origin. The release was checked through authenticated Vercel access before promotion. After promotion, five Playwright scenarios passed against the public domain: both distinct homepages, version navigation, loaded images and logo color, filters, real destinations, all three locales, mobile/tablet/desktop layout, carousel keyboard navigation and persistent theme switching. The site, V2, hero image, editorial and Hermes entry points also returned public HTTP 200 without Vercel authentication.

## Homepage and typography

The existing homepage now uses the requested [Rest of World](https://restofworld.org/) editorial arrangement: latest stories on the left, a lead story in the center and secondary stories on the right, with the lead first on phones. The additional user reference supplies the lower-page rhythm: wide photographic feature, two-story selection, newspaper promotion and dark Creata footer. After user feedback, the promotion uses a bounded cover instead of letting its portrait aspect ratio set the grid height. The footer has a restrained brand/subscription column and three navigation groups on a full-width background; its contents share the page gutters. Existing opportunities and looping carousels retain their controls. The newspaper promo reads the backend archive or the shared demo publication snapshots; withdrawing an issue removes the promotion.

Inter and Archivo are self-hosted with OFL licenses in `packages/brand/assets`. Shared typography applies to the public site, editorial and Hermes. Archivo's [official font metadata](https://github.com/google/fonts/blob/main/ofl/archivo/METADATA.pb) lists Latin subsets; [Inter](https://github.com/google/fonts/blob/main/ofl/inter/METADATA.pb) provides the Cyrillic/Kazakh coverage. Interface text uses Inter and display headings use Archivo with Inter fallback. The rendered browser font audit confirmed both actual custom fonts in mixed-script headings.

The built-in ImageGen mockup is saved at `output/imagegen/creata-news-homepage-v1.png`; its full prompt and mode are in `output/imagegen/creata-news-homepage-v1.prompt.md`. It is a design reference. The implemented articles use their existing published images and text.

The public demo uses the existing Creata frontend with explicit sample data. Enable only for demo builds with `NEXT_PUBLIC_CREATA_DEMO_MODE=1` at build time and runtime. Ordinary builds continue using the Go API; errors never silently fall back to fixtures.

The sample bundle contains articles, opportunities, fictional launches, a fictional chart, the original newspaper and original public pages. All required images/videos/PDF are portable static assets. The banner identifies the demo in RU/ҚАЗ/EN; robots prohibit indexing.

Reader preferences, saved items, votes and comments are stored separately in each visitor's browser. Existing editorial and Hermes applications run at `/editorial/today` and `/hermes/sources` on the same origin. The `@creata/demo-pipeline` adapter simulates their API and shares browser-local content with the public site. Real OAuth, AI, parsing, Telegram linking and delivery remain disabled. No production user or unpublished editorial data is exported.

Visitors can add/check/run sources, collect stories, prepare multilingual drafts, follow AI job progress, request rework, edit revisions, submit for review, approve and publish, schedule or withdraw a publication, and build/approve/simulate sending a digest. CMS pages, authors, opportunities, launches, graphics, media and newspaper operations use the same local store. Public routes read approved publication snapshots; unsaved or unapproved article/page edits do not replace them. Demo auth initially has chief editor/admin roles; logout exposes the existing role-selection login.

Each visitor's changes are isolated in their own browser. Open the three apps on the same Vercel origin to share them across tabs. `Сбросить демо` restores the sample content and clears demo reader preferences. Browser storage limits total media uploads; quota errors are shown without losing the previously saved state. Real services and credentials are unnecessary.

## Prepare and verify

1. `bun run seed:dev`, then `bun run demo:prepare` rebuild the public fixtures and media from the supplied prototype/import bundle.
2. `bun run demo:build-admins` compiles the existing editorial and Hermes Vite applications with explicit demo flags and `/editorial` / `/hermes` bases. It records only the current build assets in `.runtime/demo-admin-assets.json`.
3. Build the web workspace with `NEXT_PUBLIC_CREATA_DEMO_MODE=1`, a separate `CREATA_BUILD_DIR`, and `CREATA_API_ORIGIN=http://127.0.0.1:1` to prove backend independence. Demo-only Next rewrites preserve static assets and support SPA deep links/reloads.
4. Run `bun test packages/demo-pipeline/src creatanews/src/demo/api.test.ts`. Run `e2e/demo.spec.ts` and `e2e/admin-demo.spec.ts` with `CREATA_DEMO_TEST_URL` and `CREATA_FULL_DEMO_TEST_URL` pointing to that production build. `CREATA_BROWSER_EXECUTABLE` optionally selects an installed Playwright browser. These tests never target the working database.

## Stage and publish

`bun run demo:stage` writes a new directory under `.runtime/deploy/` and records its exact path in `.runtime/vercel-demo-stage.json`. It flattens only the frontend and the brand/contracts/graphics/demo-pipeline packages, pins tested dependency versions, and copies required fonts, demo media and manifest-listed compiled admin assets. Backend, bot, worker, archives, environment files and stale admin asset hashes are excluded. The original workspace is never uploaded.

In the reported stage directory, run `bun install` to generate its isolated lockfile. Link explicitly to `ferumms-projects/creata-news-demo`, then run `vercel project inspect --non-interactive` to confirm the target. Deploy with `--prod`, the explicit demo flag in both build/runtime, and the public site URL in `CREATA_PUBLIC_URL`. Keep `CREATA_API_ORIGIN` set to the unused local port above. Do not upload a root `.env`.

Public entry points:

- Site: https://creata-news-demo.vercel.app/ru
- Editorial: https://creata-news-demo.vercel.app/editorial/today
- Hermes: https://creata-news-demo.vercel.app/hermes/sources

Project: `ferumms-projects/creata-news-demo`. The current deployment is recorded above; the preceding release was `dpl_7qge3CLJgs7GmAYwZWYCUvV9cKn9` on 15 November 2026. The demo is an independent Vercel deployment; it does not depend on the local computer remaining online.

Verified without Vercel authentication: all three entry points return HTTP 200; all three languages, public pages/media, 36 newspaper pages and PDF, search, saved articles, profile persistence and visitor isolation, launch votes/comments, mobile layout and looping cards. All 13 browser scenarios passed in a complete run against the preceding full-pipeline release (`dpl_7AcW9QoGikC2AvQuJvYMfgr9EHyv`). The 10 baseline scenarios and 3 focused homepage scenarios also passed in local runs against a dead backend origin. They cover every editorial/Hermes sidebar section, source → collection → multilingual draft → manual edit → submit → approval → publication → public article/home/listings → simulated digest delivery → withdrawal, new opportunities/authors and their public detail links, CMS preview/publication, browser isolation, and 360/768/1440 px layouts. Added homepage checks verify three-column hierarchy, lead-first phone layout, both fonts in all three applications, carousel wrapping/keyboard/wheel, theme switching, footer fit, and the newspaper promotion's live publish/unpublish behavior. The original issue's lazy browser content has an explicit 20-second loading allowance in the test; publication assertions wait for streamed HTML to settle.

25 unit tests / 131 assertions, web/editorial/Hermes/mock-package TypeScript, both Vite production builds and Next production build passed. The mock-engine spec and quality reviews passed in the prior phase. The homepage was visually checked against both references in light/dark themes and on phones, tablets and desktop. A deployment-scoped Vercel error-level log query returned no matching logs. The existing frontend components, backend contracts and editorial pipeline remain the implementation owners.

Latest promo/footer refinement: 7 public/homepage scenarios passed locally and on production. At 1440px the promotion measures 312px instead of 674px, the footer 302px instead of 608px, and the brand 46px instead of 307px. Loaded-cover captures and geometry checks confirm no horizontal overflow and aligned gutters at 360/768/1000/1440px. The new density regression failed against the original production banner before the fix. Scoped Impeccable mechanical scan: no findings. Independent layout review passed after correcting intermediate gutters. Issue publication, reader actions and carousel controls remain covered.
