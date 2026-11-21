# Public site deployment

Build from the workspace root using `creatanews/Dockerfile`. The Next.js standalone server listens on port 3002. Set `CREATA_API_ORIGIN` to the private Go API origin and `CREATA_PUBLIC_URL` to the public HTTPS frontend origin. Browser API requests remain on the frontend origin and are forwarded by the Next.js route handler at runtime.

Deploy and migrate the new Creata database before starting the website. Add the exact public frontend origin to the API's permitted origins. Authentication is supplied by the Go API: configure Google and Telegram there. No provider secret belongs in a `NEXT_PUBLIC_` variable or this application.

Keep `CREATA_DEV_AUTH` disabled outside local development. The website does not create accounts or fabricate articles when the API is unavailable; it presents a recoverable error. Empty databases present an empty newsroom. Development fixtures are never a production fallback.

Verify a public article in server-rendered HTML, language alternatives, canonical URLs, `/robots.txt`, `/sitemap.xml`, sign-in, saved records and Telegram linking through the deployed HTTPS origin. Private preview, account and saved pages are not indexable.
