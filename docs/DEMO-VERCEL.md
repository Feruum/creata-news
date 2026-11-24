# Creata News demo

The public site, editorial app and Hermes share a browser-local demo store on one origin. The sample content includes the newspaper, articles, opportunities, launches and a chart. OAuth, AI, Telegram and publication to external services are simulated or disabled.

- [Public site](https://creata-news-demo.vercel.app/ru) and [editorial homepage](https://creata-news-demo.vercel.app/ru/v2)
- [Editorial app](https://creata-news-demo.vercel.app/editorial/today)
- [Hermes](https://creata-news-demo.vercel.app/hermes/sources)

## Local build

Run from the workspace root after `bun install --frozen-lockfile`.

```sh
NEXT_PUBLIC_CREATA_DEMO_MODE=1 CREATA_BUILD_DIR=.next-demo bun run --filter @creata/web build
NEXT_PUBLIC_CREATA_DEMO_MODE=1 CREATA_BUILD_DIR=.next-demo bun run --filter @creata/web start
```

The opt-in web build generates media and both admin bundles. To refresh the committed sample data, run `bun run seed:dev` followed by `bun run demo:prepare`.

Run `bun test packages/demo-pipeline/src creatanews/src/demo/api.test.ts` for the demo store. Point `CREATA_DEMO_TEST_URL` and `CREATA_FULL_DEMO_TEST_URL` at the same running site for `e2e/demo.spec.ts` and `e2e/admin-demo.spec.ts`.

## Vercel package

Run `bun run demo:stage` from the workspace root. It generates both admin builds and copies required public assets into a new directory under `.runtime/deploy/`. The exact path is in `.runtime/vercel-demo-stage.json`.

In that directory, run `bun install` to generate a lockfile. Link the stage to `ferumms-projects/creata-news-demo` and verify it with `vercel project inspect --non-interactive`. Use `NEXT_PUBLIC_CREATA_DEMO_MODE=1` at build time and runtime, set `CREATA_PUBLIC_URL` to the demo origin and use `CREATA_API_ORIGIN=http://127.0.0.1:1`. Deploy the stage directory with `vercel --prod`.
