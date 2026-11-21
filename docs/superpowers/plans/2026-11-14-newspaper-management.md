# Newspaper management implementation plan

**Goal:** Let the newsroom create newspaper issues, select and order published articles, upload photographs, preview and explicitly publish each issue into the public archive.

**Architecture:** Use the existing Go/PostgreSQL document store, session/CSRF checks and media upload endpoint. Store editable issues separately from immutable public snapshots. Extend the existing React admin and Next.js newspaper routes; preserve the imported issue 01.

**Tech stack:** Go, PostgreSQL, OpenAPI-generated TypeScript contracts, React/TanStack Query/Router, existing Base UI controls, Next.js.

## Tasks

- [x] Add failing API tests for draft privacy, permissions, ordered article snapshots, revision conflicts, duplicate slugs, image validation, republishing and unpublishing.
- [x] Implement `backend/creata/internal/news/newspaper.go`, register routes and add a unique issue slug index. Authors may assemble their own drafts; editors may edit drafts; chief editors and administrators publish whole issues. Keep the last published snapshot visible while a new revision is edited.
- [x] Extend OpenAPI and regenerate shared contracts. Add `events-admin/src/creata/newspaper-*` list/editor/media components using existing controls and upload endpoint, with preview and explicit publish/unpublish actions. Add the section to the existing sidebar/router and three interface languages.
- [x] Load public issues from the API, merge the imported 01 with published issues, remove the build-time-only issue allowlist, render new issues in the existing newspaper style, and add authenticated draft previews. Update legacy redirects and sitemap.
- [x] Verify API tests against isolated PostgreSQL schemas; run typechecks/builds and browser tests through an isolated local stack. Exercise real image upload, issue creation, article selection, ordering, preview, publication, archive appearance, draft isolation and unpublishing. Preserve running services used by the main conversation.

## Acceptance

A saved draft survives reload, is absent from public endpoints, and rejects stale revisions with 409. A published issue contains fixed copies of its selected articles and photos. Draft changes do not silently change the archived edition. Republishing replaces that snapshot deliberately; unpublishing removes it. New issues are available without rebuilding the site. The original imported 01 remains available.
