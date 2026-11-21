# Editorial and Hermes interactive demo

> Use subagent-driven-development for the isolated mock engine, and review the implementation before release. User authorized the existing UI and pipeline to be extended with mocks and published. No Git repository is present; changes remain in the workspace.

**Goal:** Expose the existing editorial app and Hermes console as working demo entry points linked to the existing Vercel frontend.

**Architecture:** Serve the existing Vite applications at /editorial and /hermes under the same origin as the public demo. A new browser-local mock engine implements the existing API contracts and shares state across the two apps and the public site. Preserve existing components/styles. Simulate collection, AI progress, immutable revisions, editorial approval/publication, digest preparation and delivery. Every visitor has isolated persistent demo data. No real AI, provider credentials, parsing or Telegram sends.

**Ownership:** Mock-engine implementer owns packages/demo-pipeline only. Root owns existing app adapters, browser public readers, Vite bases, deployment packaging, end-to-end verification and Vercel publication. No shared edits without coordination.

- [x] Implement/test typed browser-local API simulation and publication snapshots, including conflict/approval rules and full source→story→job→article→digest flow.
- [x] Connect editorial/Hermes demo auth and transport, existing links and base paths; preserve normal server transport.
- [x] Make public demo reads reflect published snapshots and reader-local saves, while drafts stay in preview.
- [x] Bundle both production Vite builds under the existing Next demo with safe SPA routing.
- [x] Check every sidebar section, CRUD interactions, provider-free AI simulation and source→publication→public→digest scenario, plus visitor isolation and mobile rendering.
- [x] Verify Vercel team/project, stage only approved demo files, publish and repeat checks on the public URL.

Release: `dpl_9h2zkbK2nPFXTkGm9YTCQat8RNoY`, production READY, `https://creata-news-demo.vercel.app`. Verification: 25 unit tests / 131 assertions; TypeScript for web/editorial/Hermes/mock package; both Vite builds and Next production build; 10 browser scenarios passed locally and on the public deployment. Narrow spec and quality reviews passed after corrections. Existing style sheets were preserved.
