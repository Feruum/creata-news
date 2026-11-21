# Club preview verification

From the repository root, install the frozen dependencies and generate/build
`@studyverse/proto` before running browser tests.

The fixture suite uses the real admin components with an in-process Connect
transport. It starts and stops its own server on port 6886:

```sh
node apps/events-admin/node_modules/@playwright/test/cli.js test --config apps/events-admin/playwright.club-preview.config.ts --project chromium
```

The existing club-management suite also verifies legacy admin entry points:

```sh
node apps/events-admin/node_modules/@playwright/test/cli.js test --config apps/map/playwright.clubs.config.ts --project chromium --grep admin
```

Run these suites sequentially: their fixture builders share an output directory.
On Windows the default browser channel is Edge; `PLAYWRIGHT_CHANNEL` can override it.

## Optional live integration

`playwright.club-live.config.ts` is deliberately separate from the default test
discovery. It creates clubs, role templates and memberships through the real API.
Run it only against disposable local QA services, never production data.

It expects:

- A production admin preview at `http://localhost:5173/administration/`, built with
  `VITE_API_URL=http://localhost:5555` and
  `VITE_KRATOS_PUBLIC_URL=http://localhost:14433`.
- The real backend at `localhost:5555`, backed by an isolated, migrated PostgreSQL
  database, SpiceDB and object storage. Object storage must be reachable by the browser.
- Kratos public API at `localhost:14433`, with four synthetic password identities
  synchronized to backend users: `admin`, `settings`, `members` and `reader`.
  Only `admin` has the platform administrator relation initially. The test assigns
  organization permissions to the other three identities itself.
- An ignored `apps/events-admin/test-results/club-preview-qa/accounts.json` with
  this structure (use the actual generated QA identity credentials and user IDs):

```json
{
  "admin": { "id": 1, "email": "admin@example.test", "password": "generated-test-password" },
  "settings": { "id": 2, "email": "settings@example.test", "password": "generated-test-password" },
  "members": { "id": 3, "email": "members@example.test", "password": "generated-test-password" },
  "reader": { "id": 4, "email": "reader@example.test", "password": "generated-test-password" }
}
```

```sh
node apps/events-admin/node_modules/@playwright/test/cli.js test --config apps/events-admin/playwright.club-live.config.ts --project chromium
```

The live suite checks persistence of every editable field, logo upload/removal,
nullable role fields, shared templates across clubs, membership expiry, history,
both UI permissions and forbidden RPC responses, and inline save errors with retry.
Service workers are blocked in this suite so its deliberate failed-RPC interception
reaches the production QueryClient; PWA behavior has its own project checks.
Traces and screenshots are
written under ignored `test-results/` directories. Keep runtime credentials,
service configuration and QA database artifacts out of commits.
