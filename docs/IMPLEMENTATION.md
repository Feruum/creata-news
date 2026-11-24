# Creata News implementation

Creata News is an independent RU/kk/en news platform. The development seed reads the prototype data fixture in `scripts/fixtures/prototype-seed.js`.

## User visual correction — 14 November 2026

Preserve the existing frontend and admin interface owners. The public app reuses the original Shell, Spotlight, poster rail, carousel and styles from the archive; news data is adapted into those components. Admins reuse the original Sidebar and shadcn/Base UI primitives. A new wholesale interface is not authorized. One shared Creata identity (violet #4B44F0, original local Manrope/Kazakh fonts, common neutrals, logo and controls) replaces the earlier separate pink admin theme. The public header has one responsive primary navigation and one disclosure menu, with no stacked utility/category/menu bars. Rest of World and Skiper supplement the existing interface.

Opportunity cards and the countdown are adapted from the original prototype. `creatanews/PROTOTYPE-SOURCE-REUSE.json` records their origins. The API stores the original photo and localized audience. The development seed contains five approved opportunities, including the Creata Games hackathon, and the “Mama Pro” hackathon awaiting editorial review.

The home opportunity carousel uses the existing Embla/CarouselActions components, loops through all approved records and shows four/two/one cards at desktop/tablet/mobile widths. It advances every six seconds while visible, pauses on hover, focus, dragging and hidden tabs, and disables autoplay for reduced motion. Manual arrows, keyboard and touch remain available. The public list refreshes every minute and on returning to the tab; new approved opportunities appear without source changes.

## Ownership and flow

Go API owns new PostgreSQL schema, authorization, editorial revisions and public snapshots, source/job leases, budgets, digests/deliveries and analytics. Python Hermes is a restricted collection/classification/drafting worker and cannot publish. Bun/Grammy bot only manages subscriptions and authorized deliveries. Next renders public articles; React/Vite apps own editorial and Hermes controls.

Source -> collect/normalize/deduplicate -> story bank -> classify -> draft/research/review/three locales -> human editorial approval -> immediate/scheduled publication -> digest selection/approval -> delivery -> analytics.

API contract: API-CONTRACT.md and backend/creata/openapi.json. Generated TypeScript schemas: packages/contracts/src/generated.ts.

## Release gates

- Clean workspace installation, typechecks and builds.
- New database only; explicit opt-in development seed.
- Published snapshot isolation and version conflict tests.
- RBAC, worker capabilities, secret isolation, CSRF and source-network protections.
- Collection, unconfigured AI, budget reservations, retry/leases/dedup integration.
- Reader language/profile/save/link flow and editorial publish flow in browser.
- Bot delivery state tests including 429 and ambiguous outcomes.
- Live AI and Telegram smoke tests only after operator credentials/configuration.

Postiz account selection and delivery are implemented for approved article and chart revisions. [Operational pipeline](PIPELINE.md)

Deferred: email, video generation, print PDF, interactive builder, ML and social community.
