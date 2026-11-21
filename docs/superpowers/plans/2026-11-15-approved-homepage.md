# Approved Creata homepage implementation plan

**Goal:** Implement the user-approved full-page image with the project's existing CR≡ATA NEWS identity and violet/neutral tokens.
**Architecture:** Extend HomeScreen, HomeFeatures and the existing home-only Shell branches. Reuse ArticleCard, CreataBrand, Embla useCarousel, the native menu dialog, locale routing, published issue data and existing campaign photography. Do not introduce new dependencies or change API/data ownership.
**Stack:** Next 16 / React 19, CSS Modules and scoped global CSS, Inter/Archivo, Playwright (installed Chrome).
**Approved reference:** C:/Users/Ferumm/.codex/generated_images/01a1df97-5cd0-7f83-8554-fb081a254be1/exec-64ce3b6c-4ea0-42ba-a453-ab60579a537c.png

## Tasks
- [x] Establish the current source baseline and focused browser assertions and retain the existing content/locale/theme/menu/carousel behavior.
- [x] Update home-only navigation and footer in creatanews/src/components/shell.tsx; replace accumulated home rules in creatanews/src/shared/styles/editorial-home.css. The header is one compact row with existing logo, four editorial links, search/newspaper actions and reachable preferences/menu. Footer uses the large original wordmark and real digest link, with no nonfunctional email field.
- [x] Align creatanews/src/features/news/components/home-screen.module.css and home-screen.tsx with the violet/dark hero, six editorial cards and purple active filters. Update home-features.tsx with existing campaign portrait, two editorial monthly cards and dark newspaper promotion. Keep published issue href/cover and empty/unpublished handling. Update home-copy.ts for approved copy across ru/kk/en and replace decorative author asterisks with the brand triple bar in cards.tsx.
- [x] Verify typecheck, relevant existing unit tests, production build, and browser flows on ru/kk/en at desktop/tablet/mobile widths. Exercise topic filters, theme toggle, menu close/focus, carousel keyboard, newspaper and digest destinations. Inspect full-page screenshots against the approved image; document real-content adaptations.

## Verification
Use a separate local demo server on port 3005 with NEXT_PUBLIC_CREATA_DEMO_MODE=1 and CREATA_BUILD_DIR=.next-approved-home to avoid interfering with existing servers. Data comes from the project's explicit public demo fixtures; production continues to use the API. Browser checks must detect page errors, failed image loads, horizontal overflow, clipped branding and broken navigation. Article imagery and newspaper cover remain CMS-owned, rather than substituting generated article content.

## Completed verification
- `bun run typecheck`: passed.
- `bun test src`: 21 passed, 472 assertions.
- Production build with demo mode disabled: passed.
- Focused Playwright suite: 4 passed; ru/kk/en, 360/768/1000/1440px, filters, loaded images, theme persistence, native menu focus, mobile and expandable popular carousels, locale switch, search/newspaper/profile destinations.
- The approved light theme is the default; explicit reader preferences remain persistent.
- Article imagery and the latest published issue cover are intentional content-driven adaptations of the design reference.


