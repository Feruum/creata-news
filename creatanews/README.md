# Creata News — public website

Next.js application for the Creata News newsroom. The public site reads published snapshots from the Go API; account actions use the same API through a runtime server proxy. No news, authentication tokens or article edits are stored in browser localStorage. Only the visual theme preference is local.

From the workspace root:

```sh
bun install --frozen-lockfile
bun run --filter @creata/web dev
```

The site runs at `http://localhost:3002`. Configure `CREATA_API_ORIGIN` (default `http://127.0.0.1:5555`) and `CREATA_PUBLIC_URL` (`CREATA_SITE_URL` is accepted as a legacy alias) (canonical public origin) in the app environment. API configuration is read at runtime, including in Docker.

Routes are prefixed by `/ru`, `/kk` or `/en`: home, world, ecosystem, articles, categories, authors, opportunities, search, saved and profile. The root returns to the last selected language. Editorial previews at `/{locale}/preview/{id}` require an authenticated staff session and are excluded from indexing.

```sh
bun run --filter @creata/web typecheck
bun run --filter @creata/web test
bun run --filter @creata/web test:e2e
```

Browser tests use the running site and API with the explicit development seed. The save scenario additionally requires explicitly enabled local development authentication. Configure `CREATA_WEB_URL` to test a different origin and `PLAYWRIGHT_CHANNEL` to select an installed browser.

Homepage: the approved Creata editorial design uses a compact header, violet/charcoal hero, three-column story grid, full-width portrait, monthly stories, dark newspaper promotion and oversized CR≡ATA NEWS footer. Article images, text and newspaper covers come from the existing published data. The original locale routing, native menu, theme switch, Embla carousel hook and CarouselActions are reused. Opportunities and popular stories remain accessible in expandable sections below the newspaper banner. Phones use the existing keyboard- and touch-accessible story carousel.

`@creata/brand` supplies the original logo, violet/neutral tokens and self-hosted Inter/Archivo fonts. The default is the approved light palette. Readers can save a dark theme preference. The editorial homepage artwork is in `public/images/home/creata-human-hero-v1.png`.

Homepage motion uses CSS for first-load, hover and press feedback, Motion's `inView` for once-per-entry editorial reveals, and a 180ms native Web Animations fade for pointer-driven topic changes. Reveals animate card children, preserving Embla's slide transforms. Keyboard filtering stays immediate; reduced-motion and coarse-pointer preferences suppress movement. Content remains visible without the motion controller. `e2e/home-motion.spec.ts` covers interruption, repeated scrolling, keyboard input and reduced motion.

The opportunities section adapts `OppsBand`, `OppCard` and `Countdown` from the original prototype. Its scoped styles are in `src/features/opportunities/prototype-card.css`. `PROTOTYPE-SOURCE-REUSE.json` records the original fragments and hashes. Cards use published opportunities from the API and readers can save them to their accounts.

The homepage extends that block with a looping carousel: four cards on desktop, two on tablets, one on phones. It supports touch drag, keyboard arrows, the restored CarouselActions and a pause control. Autoplay pauses for hover, focus and hidden tabs, and stays off with reduced motion. All approved API records are included; updates arrive every minute and when the tab receives focus. A failed refresh preserves loaded cards and offers retry.
