# Internal notification HTTP API

Set the same nonempty `INTERNAL_REMINDER_SECRET` in the backend and Telegram bot. Staging and production require at least 32 characters. Backend domain-event notifications and the administrative notification RPC send it as `X-Internal-Secret`.

`POST /notify` and `POST /notify/bulk` reject missing or incorrect credentials before reading the request body. An empty configured secret disables these routes. Requests require JSON and are limited to 2 MiB, messages to 1–4096 UTF-16 code units, and bulk requests to 1–100 notifications. Inline keyboards also have bounded rows, buttons, labels, URLs, and callback data. Invalid bodies do not send messages. Telegram delivery failures return HTTP 502 so durable backend delivery can retry.

The backend validates messages and sends larger recipient lists in batches of at most 100. Delivery is at least once: a failed batch can contain recipients who already received the message before another recipient failed.

The public Telegram webhook retains its separate `X-Telegram-Bot-Api-Secret-Token` credential. Health endpoints remain public.
