# Browser-local editorial and Hermes demo

`@creata/demo-pipeline` implements the existing Creata API shapes in a visitor's browser. It never calls `fetch`, starts a worker, contacts an AI provider, fetches source websites, or sends Telegram messages. Text, research, classification, budgets, analytics and deliveries are simulations. The existing applications supply the seed JSON; this package imports no application source.

```ts
import { createPipelineDemo, DEMO_PIPELINE_EVENT } from '@creata/demo-pipeline'

const demo = createPipelineDemo(seed)
const session = await demo.request('/auth/me')
const articles = demo.getPublishedArticles()
const article = demo.getArticle('id-or-slug') // published snapshots only
const workingState = demo.getSnapshot() // detached copy, includes drafts
window.addEventListener(DEMO_PIPELINE_EVENT, refreshQueries)
// Also listen for the standard `storage` event for updates from other tabs.
```

`createPipelineDemo(seed, storage?, now?)` accepts `DemoSeed`: required `articles`, `categories`, `opportunities`, and optional `authors`, `launches`, `pages`, `newspapers`. The optional storage is native Storage-compatible (`getItem`/`setItem`) or `Map<string,string>`; the optional clock returns an epoch millisecond number or Date. All types, `DEMO_PIPELINE_KEY` (`creata:pipeline-demo:v1`), and `DEMO_PIPELINE_EVENT` (`creata:pipeline-demo:change`) are exported from the package. `reset()` replaces local state with the engine's initial seed.

Default browser storage is `window.localStorage`, with a memory fallback when access is unavailable from the start. Outside browsers, each instance gets a fresh memory store. Same-origin applications and tabs use the same storage key. No shared mutable server state is needed. Reads reload current storage, add only missing seed items/fields, preserve local edits and deletion markers, and return detached copies. Reads never write storage or dispatch change events. Successful mutations persist state and emit one custom change event. A runtime persistence failure rejects the mutation with `ApiError(507)` for quota exhaustion or `ApiError(503)` for other storage failures, preserves the last committed state, and emits no change event. Repeated long HTML/data URLs are pooled in storage without changing the decoded state shape.

Working `articles` and immutable `publications: Record<articleId, Article>` are distinct. Edits increase the content revision and invalidate approval. Approval binds that exact revision; publish requires a matching approval, and a stale `expectedRevision` raises `ApiError(409)`. Rework is local simulated AI work and refuses to overwrite a later manual edit. Unpublish removes the public snapshot. Scheduled publication and job progress are projected deterministically from the clock on reads, then persisted with the next mutation. No timers need to keep running when the app closes.

`pagePublications` uses page slugs (`about`, `ecosystem`, `newspaper-01`) as keys; `newspaperPublications` uses issue IDs. These are independent publication records. CMS edits keep the previous snapshot until publish. Newspaper snapshots contain copies of the published articles selected at the time of issue publication. `pages`/`newspapers` contain working drafts; `pageRevisions`/`revisions` contain history.

Default auth has chief editor and admin roles, and its user ID matches an author. `/auth/logout` and `/auth/dev-login` let the existing login screen simulate `reader`, `author`, `editor`, `chief_editor`, and `admin`. Role checks apply to mutations. Provider login and real Telegram linking explicitly return 501. AI settings discard entered keys and always retain the demo model/endpoint. The HTTPS `.invalid` URL keeps the existing URL form field valid and is never contacted.

The supported UI routes cover sources/check/run, manual URL collection, bank/evidence/merge/split, jobs/retry/cancel/fail, drafts/graphics/rework/research, revisions and editorial actions, authors/opportunities/launches, staff/users, settings/rules, digests and simulated delivery, analytics/audit, CMS pages, media, and newspaper editing/publication/preview. Unknown verbs or paths raise `ApiError`. A run spends 1 second queued and progresses through classification/research/draft/review, completing after 10 seconds. Repeated queue clicks and job retries do not duplicate active work or collected stories.

Uploads accept JPEG/PNG/WebP/GIF/AVIF, up to 2 MiB per file, and return browser-local data URLs with an explicit demo rights annotation. File IO finishes before reading the current mutation state, so an upload cannot overwrite edits committed while its file was being read. Used assets cannot be deleted. Browser storage capacity still limits the total amount of uploaded media; exceeding it reports an unsaved upload rather than silently switching to memory. The temporary fallback for initially inaccessible storage cannot survive a full browser reload.

Validation:

```sh
bun test packages/demo-pipeline/src/index.test.ts
bun x tsc --noEmit -p packages/demo-pipeline/tsconfig.json
```

The suite covers the complete source → collect → draft → review → approve → publish → digest flow, role checks, conflicts, frozen snapshots, retry idempotency, per-storage isolation/restart, seed upgrades/deletions, read-only projections, scheduling, CMS publication, uploads, and restricted storage.
