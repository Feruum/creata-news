export type SigNozInfo =
  | { configured: false }
  | { configured: true; url: string; login?: { email: string; password: string } }

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isRootUrl(value: unknown): value is string {
  if (typeof value !== 'string' || !/^https:\/\/[^/\\?#\s]+\/?$/.test(value)) return false
  try {
    const url = new URL(value)
    return (
      url.protocol === 'https:' &&
      !!url.hostname &&
      !url.username &&
      !url.password &&
      url.pathname === '/' &&
      !url.search &&
      !url.hash
    )
  } catch {
    return false
  }
}

function parseInfo(value: unknown): SigNozInfo {
  if (!isRecord(value)) throw new Error('Invalid SigNoz access')
  if (value.configured === false && Object.keys(value).length === 1) return { configured: false }
  if (
    value.configured !== true ||
    !isRootUrl(value.url) ||
    Object.keys(value).some((key) => !['configured', 'url', 'login'].includes(key))
  ) {
    throw new Error('Invalid SigNoz access')
  }
  if (!Object.hasOwn(value, 'login')) return { configured: true, url: value.url }
  const login = value.login
  if (
    !isRecord(login) ||
    Object.keys(login).length !== 2 ||
    typeof login.email !== 'string' ||
    !login.email.trim() ||
    typeof login.password !== 'string' ||
    !login.password.trim()
  ) {
    throw new Error('Invalid SigNoz access')
  }
  return { configured: true, url: value.url, login: { email: login.email, password: login.password } }
}

export async function loadSigNozAccess(signal: AbortSignal): Promise<SigNozInfo> {
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:5555'
  const response = await fetch(`${apiUrl.replace(/\/$/, '')}/admin/signoz/info`, {
    method: 'GET',
    credentials: 'include',
    cache: 'no-store',
    headers: { 'X-Studyverse-SigNoz': '1' },
    signal
  })
  if (!response.ok) throw new Error('Unable to load SigNoz access')
  try {
    const body: unknown = await response.json()
    return parseInfo(body)
  } catch {
    throw new Error('Unable to load SigNoz access')
  }
}
