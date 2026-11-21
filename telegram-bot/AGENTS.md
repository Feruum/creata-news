# Telegram Bot - Grammy

**Parent:** `../../AGENTS.md`

Grammy-based Telegram bot for AITU schedule and notifications.

## Entry Point

`src/index.ts`:
```typescript
startServer()                    // HTTP server (health, webhooks)
setupCommands(bot)               // Register command handlers
setupCallbacks(bot)              // Register callback query handlers
bot.api.setMyCommands([...])     // Register menu commands
startScheduler(bot)              // Cron jobs (Croner)
startEventReminderScheduler(bot) // Event reminder cron
bot.start()                      // Launch polling (try/catch — HTTP stays up on failure)
```

## Architecture

```
src/
├── commands/         # /start, /settings, /schedule, /events, /reminders, /test, /help
├── callbacks/        # Inline keyboard handlers
├── features/         # Domain modules (events, reminders)
├── services/         # Service layer (events, map, schedule, scheduler)
├── server/           # HTTP server (health endpoint, webhooks)
├── lib/              # Shared utilities
├── db/               # Drizzle schema (separate from packages/db)
├── bot.ts            # Bot instance creation
└── index.ts          # Entry point
```

## Adding Commands

1. Create handler in `src/commands/`
2. Register in `setupCommands()` in `commands/index.ts`
3. Add to `setMyCommands()` array in `index.ts` if user-facing

## Adding Scheduled Tasks

1. Create job in `src/services/scheduler.ts` or `src/features/reminders/`
2. Register in startup flow in `index.ts`
3. Use Croner syntax for schedule

## Key Dependencies

| Package | Purpose |
|---------|---------|
| grammy | Telegram Bot API |
| croner | Cron scheduling |
| drizzle-orm | Database ORM |

## Commands

```bash
bunx nx dev @studyverse/telegram-bot   # Dev
bunx nx build @studyverse/telegram-bot # Build
```

## Database

Own Drizzle schema at `src/db/schema.ts`. Separate from web apps (`packages/db`).
