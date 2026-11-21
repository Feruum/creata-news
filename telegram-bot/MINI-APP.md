# Campus map in Telegram

The bot opens the existing map website as a Telegram Mini App. The site keeps its existing account sign-in; Telegram launch data is used only for host presentation and is not an authenticated identity.

## Deployment

1. Deploy the map and bot changes together.
2. Set `MAP_APP_URL` to the public HTTPS map origin, normally `https://map.webspace.cat`. The VPS compose file already supplies this value.
3. At startup the bot sets the default menu button to **Открыть карту** through `setChatMenuButton`. `/start` and `/map` also refresh the private chat's menu, replacing older per-chat command menu overrides. No BotFather change is required for this button.
4. In a private bot chat, use `/start` or `/map`, then open the map. Room and event notification buttons open their corresponding site routes.

Web App buttons are available in private chats. Group chats use regular website links. When `MAP_APP_URL` uses HTTP for local development, the bot retains the command menu and uses regular links; test an actual Mini App with a public HTTPS development URL.

Event reminders offer **Open event to cancel registration**. The user signs into the existing site account and cancels on the event screen. Telegram callback data never authorizes a cancellation. Older cancellation buttons show instructions and launch the event (or the app home if the event cannot be resolved).

Set the same nonempty `INTERNAL_REMINDER_SECRET` on the backend and bot for internal notifications. The `/notify` and `/notify/bulk` endpoints require this value in `X-Internal-Secret`; bulk requests accept at most 100 recipients. Lesson reminder failures retry from 15 minutes until one minute before class; abandoned claims become eligible after two minutes. Sent delivery records are never retried.

## Profile launch button

To add Telegram's **Launch app** button to the bot profile and enable `https://t.me/<bot_username>?startapp`, open **BotFather → /mybots → @aitumap_alerts_bot → Bot Settings → Configure Mini App → Enable Mini App**, then set `https://map.webspace.cat/`. The Bot API token can configure chat menus but cannot enable this separate profile setting. Verify `getMe.has_main_web_app` after enabling it.

## Website behavior

- Loads the official Telegram Web App SDK.
- Calls `ready()` and expands the window; disables vertical minimization swipes on supported clients so map panning remains usable.
- Follows Telegram's light/dark theme without overwriting the saved browser preference.
- Uses stable viewport height and safe areas for the map, floating controls and dialogs.
- Native Back first dismisses a site dialog. On a directly opened detail page, it returns to the map instead of navigating outside the app.
- Normal browser use and the existing account authentication flow remain available.

Reference: [Telegram Mini Apps](https://core.telegram.org/bots/webapps).
