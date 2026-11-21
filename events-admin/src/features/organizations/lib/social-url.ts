export function organizationSocialUrl(value: string | undefined, platform = 'website') {
  if (!value?.trim()) return undefined
  const trimmed = value.trim()
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  if (/^[a-z][a-z\d+.-]*:/i.test(trimmed)) return undefined
  const prefixes: Record<string, string> = {
    instagram: 'https://instagram.com/',
    telegram: 'https://t.me/',
    tiktok: 'https://www.tiktok.com/@'
  }
  return `${prefixes[platform] || 'https://'}${trimmed.replace(/^@/, '')}`
}
