# Creata Editorial and Hermes Control

`src/main.tsx` starts Editorial. Hermes Control starts in `../agent-admin/src/main.tsx` and imports the same application from `src/creata/app.tsx`. Check both modes when changing shared routes, navigation, or UI.

- Put newsroom features in `src/creata` and reuse the components in `src/shared`.
- Use kebab-case filenames and keep hooks in `.ts` files. The `@/` alias points at `events-admin/src` in both apps.
- Run `bun run --filter @creata/editorial test` from the repository root. Use `bun run --filter @creata/editorial test:e2e` for browser coverage in `e2e-creata`.

See the [Editorial README](README.md) for local setup.
