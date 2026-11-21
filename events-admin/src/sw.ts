/// <reference lib="webworker" />
import { cacheNames, clientsClaim, setCacheNameDetails } from 'workbox-core'
import { createHandlerBoundToURL, precacheAndRoute } from 'workbox-precaching'
import { NavigationRoute, registerRoute } from 'workbox-routing'
import { NetworkOnly } from 'workbox-strategies'

const worker = globalThis as unknown as ServiceWorkerGlobalScope
const base = import.meta.env.BASE_URL
const scope = new URL(base, worker.location.origin)
const cacheSuffix = scope.pathname.replace(/\W/g, '_')

setCacheNameDetails({ prefix: 'studyverse-admin', suffix: cacheSuffix })
clientsClaim()

// Auth, Connect-RPC, API and telemetry requests are never placed in Cache Storage.
// Precache handles only build assets. There is no runtime data or image cache.
registerRoute(
  ({ request, url }) =>
    url.origin !== worker.location.origin ||
    (request.mode !== 'navigate' &&
      /\/(?:api|auth|sessions|self-service|v1\/traces|v1\/metrics|v1\/logs)(?:\/|$)/.test(url.pathname)) ||
    /\/[\w.]+Service\//.test(url.pathname) ||
    request.method !== 'GET',
  new NetworkOnly({ fetchOptions: { cache: 'no-store' } }),
  'GET'
)
registerRoute(() => true, new NetworkOnly({ fetchOptions: { cache: 'no-store' } }), 'POST')
precacheAndRoute((self as unknown as { __WB_MANIFEST: { url: string; revision: string | null }[] }).__WB_MANIFEST)
worker.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) =>
        Promise.all(
          names
            .filter(
              (name) =>
                name.startsWith('studyverse-admin-') && name.endsWith(cacheSuffix) && name !== cacheNames.precache
            )
            .map((name) => caches.delete(name))
        )
      )
  )
})

registerRoute(
  new NavigationRoute(createHandlerBoundToURL(`${base}index.html`), {
    allowlist: [new RegExp(`^${scope.pathname.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`)],
    denylist: [/\/(?:api|sessions|self-service)(?:\/|$)/]
  })
)

worker.addEventListener('message', (event) => {
  if (event.data?.type === 'SKIP_WAITING') void worker.skipWaiting()
})
