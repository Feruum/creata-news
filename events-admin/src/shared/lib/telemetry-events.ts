export type ProductEvent =
  | 'page_view'
  | 'search'
  | 'floor_change'
  | 'entity_open'
  | 'schedule_open'
  | 'event_registration'
  | 'admin_action'
  | 'theme_change'
  | 'pwa_install'
  | 'offline_open'

const attributeValues = {
  surface: ['sidebar', 'map', 'schedule', 'admin', 'search', 'page'],
  category: ['room', 'event', 'organization', 'booking', 'user', 'schedule', 'other'],
  source: ['system', 'light', 'dark', 'telegram', 'my-du', 'group', 'browser', 'standalone', 'mini-app'],
  outcome: ['started', 'success', 'error'],
  floor: ['1', '2', '3'],
  action: ['register', 'unregister']
} as const

export type ProductEventAttributes = {
  [Key in keyof typeof attributeValues]?: (typeof attributeValues)[Key][number]
}

export function safeProductAttributes(attributes: Record<string, unknown>): Record<string, string> {
  const safe: Record<string, string> = {}
  for (const [key, allowed] of Object.entries(attributeValues)) {
    const value = attributes[key]
    if (typeof value === 'string' && (allowed as readonly string[]).includes(value)) safe[key] = value
  }
  return safe
}

export function sanitizeTelemetryUrl(value: string): string {
  try {
    const url = new URL(value, 'http://localhost')
    const pathname = url.pathname
      .replace(/\/(users|rooms|events|organizations|threads|messages)\/[^/]+/g, '/$1/:id')
      .replace(/\/\d+(?=\/|$)/g, '/:id')
      .replace(/\/[a-f\d]{8}-[a-f\d-]{27,}(?=\/|$)/gi, '/:id')
    return `${url.origin}${pathname}`
  } catch {
    return '[invalid URL]'
  }
}

export function sanitizeErrorStack(stack: string | undefined, message?: string): string {
  let frames = stack ?? ''
  const messageOffset = message ? frames.indexOf(message) : -1
  const firstLineEnd = frames.indexOf('\n')
  if (messageOffset >= 0 && (firstLineEnd < 0 || messageOffset <= firstLineEnd)) {
    frames = frames.slice(messageOffset + (message?.length || 0))
  }
  return frames
    .split('\n')
    .filter((line) => /^\s+at\s/.test(line) || /^[^@\n]*@https?:\/\//.test(line))
    .slice(0, 10)
    .map((line) => line.replace(/https?:\/\/[^\s)]+/g, sanitizeTelemetryUrl))
    .join('\n')
    .slice(0, 2048)
}

const safeStringAttributes = new Set([
  'http.method',
  'http.request.method',
  'rpc.system',
  'rpc.service',
  'rpc.method',
  'db.system',
  'db.operation',
  'db.operation.name',
  'server.address',
  'network.protocol.name',
  'network.protocol.version',
  'error.type',
  'exception.type'
])

export function sanitizeSpanAttributes(attributes: Record<string, unknown>): Record<string, string | number | boolean> {
  const safe: Record<string, string | number | boolean> = {}
  for (const [key, value] of Object.entries(attributes)) {
    if (
      /(header|cookie|token|authorization|password|enduser|user\.|query|statement|message|body|payload|baggage)/i.test(
        key
      )
    )
      continue
    if ((key === 'http.url' || key === 'url.full') && typeof value === 'string') {
      safe[key] = sanitizeTelemetryUrl(value)
    } else if (key === 'http.route' && typeof value === 'string') {
      safe[key] = new URL(sanitizeTelemetryUrl(value)).pathname
    } else if (safeStringAttributes.has(key) && typeof value === 'string') {
      safe[key] = value.slice(0, 160)
    } else if (typeof value === 'number' && Number.isFinite(value)) {
      safe[key] = value
    } else if (typeof value === 'boolean') {
      safe[key] = value
    }
  }
  return safe
}
