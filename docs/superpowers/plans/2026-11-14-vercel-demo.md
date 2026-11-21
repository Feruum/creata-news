# Creata frontend demo on Vercel

> Execute with executing-plans. User authorized a public frontend demo with mock data. Preserve the existing Creata UI. No Git repository is available; stage a strict allowlist for deployment.

**Goal:** Publish a backend-independent, clearly labeled demo on a separate Vercel project.

**Architecture:** Explicit NEXT_PUBLIC_CREATA_DEMO_MODE=1 enables public fixtures and browser-local reader actions. Normal API behavior remains the default. No backend, worker, bot or real external authentication/delivery is published.

- [x] Create portable mock content and public media.
- [x] Add typed public and browser-local demo adapters; test filtering, missing resources and visitor isolation.
- [x] Connect existing API/auth/CMS entry points behind the flag and show a localized demo notice.
- [x] Verify types, tests, production build and browser interactions with the backend unavailable.
- [x] Stage frontend and required packages/assets without secrets, verify Vercel target and deploy.
- [x] Verify the public URL without account authentication and report demo limits.
