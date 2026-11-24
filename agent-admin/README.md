# Hermes Control

`src/main.tsx` starts Hermes Control using the shared application in `../events-admin/src/creata`.

Run from the workspace root after `bun install`:

```sh
bun run --filter @creata/hermes-admin dev
bun run --filter @creata/hermes-admin build
```

The development server proxies `/api` to `CREATA_API_ORIGIN` (default `http://localhost:5555`). Cross-application links use `VITE_WEB_URL`, `VITE_EDITORIAL_URL`, `VITE_HERMES_ADMIN_URL`. Docker builds receive those values as build arguments. Static nginx listens on port 80, proxies `/api` to `backend:5555`, and uses SPA navigation fallback.

Sign-in providers come from `/auth/me`; staff roles and permissions are enforced by the API. The local development role selector only appears if the backend explicitly reports `devAuth: true`. The Hermes control panel requires the administrator role. No production authentication fallback is provided.

Run the shared domain tests with `bun run --filter @creata/editorial test`. Real-browser smoke tests for editorial creation/save/approval/publication and inactive source creation run with `bun run --filter @creata/editorial test:e2e` against the running local API and applications. Set `CREATA_BROWSER_EXECUTABLE` to the installed Chromium executable if using a different browser installation. These tests require an explicitly disposable development database and enabled local development sign-in.

## Current visual implementation

The current news workflows use the retained events-admin SidebarProvider, SidebarInset, AppSidebar, NavMain, NavDocuments, SiteHeader, Button, Input, Textarea and Table component owners. Their data and navigation are adapted to Creata News. Shared identity, local original fonts, light/dark colors and the logo come from @creata/brand; the former pink dashboard and marketing welcome hero are removed.
