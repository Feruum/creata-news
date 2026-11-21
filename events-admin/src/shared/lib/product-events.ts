import type { ProductEventAttributes } from './telemetry-events'

type Category = NonNullable<ProductEventAttributes['category']>

function category(name: string): Category {
  if (/booking/i.test(name)) return 'booking'
  if (/organization|club|department/i.test(name)) return 'organization'
  if (/user|teacher/i.test(name)) return 'user'
  if (/room/i.test(name)) return 'room'
  if (/schedule|calendar|deadline/i.test(name)) return 'schedule'
  if (/event/i.test(name)) return 'event'
  return 'other'
}

export function pageCategory(path: string): Category {
  const normalized = path.split(/[?#]/)[0].replace(/^\/administration(?=\/|$)/, '')
  return category(normalized.split('/').filter(Boolean)[0] ?? '')
}

export function mutationCategory(key: readonly unknown[] | undefined): Category | null {
  const names: string[] = []
  for (const value of key ?? []) {
    if (typeof value === 'string') names.push(value)
    else if (typeof value === 'object' && value !== null) {
      const descriptor = value as Record<string, unknown>
      if (typeof descriptor.serviceName === 'string') names.push(descriptor.serviceName)
      if (typeof descriptor.methodName === 'string') names.push(descriptor.methodName)
    }
  }
  if (
    !names.some((name) =>
      /^(create|update|delete|register|cancel|approve|reject|add|remove|assign|invite|revoke|publish|archive)/i.test(
        name
      )
    )
  )
    return null
  return category(names.join('/'))
}
