import type { ProductEvent, ProductEventAttributes } from './telemetry-events'

export type { ProductEvent, ProductEventAttributes } from './telemetry-events'

const enabled = import.meta.env.VITE_OTEL_ENABLED === 'true'
let initialized = false
let record: ((name: ProductEvent, attributes: ProductEventAttributes) => void) | undefined
const pending: { name: ProductEvent; attributes: ProductEventAttributes }[] = []

export function trackProductEvent(name: ProductEvent, attributes: ProductEventAttributes = {}): void {
  if (!enabled || typeof window === 'undefined') return
  if (record) record(name, attributes)
  else if (pending.length < 64) pending.push({ name, attributes })
}

export async function initTelemetry(): Promise<void> {
  if (!enabled || initialized || typeof window === 'undefined') return
  initialized = true
  try {
    const { startBrowserTelemetry } = await import('./telemetry-browser')
    record = startBrowserTelemetry({
      tracesEndpoint: import.meta.env.VITE_OTEL_ENDPOINT || '/telemetry/v1/traces',
      apiUrls: [import.meta.env.VITE_API_URL || 'http://localhost:5555'],
      serviceName: import.meta.env.VITE_OTEL_SERVICE_NAME || 'ems-frontend',
      serviceVersion: import.meta.env.VITE_OTEL_SERVICE_VERSION || 'dev',
      environment: import.meta.env.VITE_OTEL_ENVIRONMENT || 'development',
      app: 'events-admin'
    })
    for (const event of pending.splice(0)) record(event.name, event.attributes)
  } catch {
    pending.length = 0
  }
}
