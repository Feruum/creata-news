export interface HermesAccessInfo {
  dashboardUrl: string
}

export class HermesAccessError extends Error {
  readonly reason: 'denied' | 'unavailable'

  constructor(reason: 'denied' | 'unavailable') {
    super('Hermes access unavailable')
    this.reason = reason
  }
}

function dashboardUrl(value: unknown): value is string {
  if (value === 'https://auth.webspace.cat/hermes/') return true
  if (typeof value !== 'string' || !/^http:\/\/(?:localhost|127\.0\.0\.1|\[::1\])(?::\d+)?\/hermes\/$/.test(value)) {
    return false
  }
  try {
    const url = new URL(value)
    return url.pathname === '/hermes/' && !url.username && !url.password && !url.search && !url.hash
  } catch {
    return false
  }
}

function parseInfo(value: unknown): HermesAccessInfo {
  if (typeof value !== 'object' || value === null || !('claimed' in value) || !('dashboard_url' in value)) {
    throw new HermesAccessError('unavailable')
  }
  if (typeof value.claimed !== 'boolean' || !dashboardUrl(value.dashboard_url)) {
    throw new HermesAccessError('unavailable')
  }
  // The launcher retains no integrations, incident content, credentials or OAuth data.
  if (!value.claimed) throw new HermesAccessError('denied')
  return { dashboardUrl: value.dashboard_url }
}

export async function loadHermesAccess(signal: AbortSignal): Promise<HermesAccessInfo> {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:5555'
  try {
    const response = await fetch(`${apiUrl.replace(/\/$/, '')}/admin/hermes/info`, {
      method: 'GET',
      credentials: 'include',
      cache: 'no-store',
      headers: { 'X-Studyverse-Hermes': '1' },
      signal
    })
    if (!response.ok) {
      throw new HermesAccessError(response.status === 401 || response.status === 403 ? 'denied' : 'unavailable')
    }
    const body: unknown = await response.json()
    return parseInfo(body)
  } catch (error) {
    if (error instanceof HermesAccessError) throw error
    throw new HermesAccessError('unavailable')
  }
}
