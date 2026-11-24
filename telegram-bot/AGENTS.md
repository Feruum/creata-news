# Creata Telegram bot

`src/creata/main.ts` starts the Grammy bot. Command handling is in `src/creata/bot.ts` and approved digest delivery is in `src/creata/delivery.ts`.

- Run `bun run --filter @creata/telegram-bot test` and `bun run --filter @creata/telegram-bot build` from the repository root.
- Use [the example environment](.env.creata.example) for local settings and the [root README](../README.md#telegram-delivery) for Telegram setup.
