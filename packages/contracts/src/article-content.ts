import type { schemas } from './generated'

export function safeExternalUrl(value: string): string | null {
  try {
    const url = new URL(value)
    return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password ? url.toString() : null
  } catch {
    return null
  }
}

export function safeArticlePhotoUrl(value: string): string | null {
  const href = safeExternalUrl(value)
  if (!href) return null
  const { protocol, hostname, port, pathname, search, hash } = new URL(href)
  return protocol === 'https:' && ['upload.wikimedia.org', 'thumb.wikimedia.org'].includes(hostname) &&
    !port && !search && !hash &&
    pathname.startsWith('/wikipedia/commons/') ? href : null
}

export function articlePhotoLayout({ media, photos }: Pick<schemas['Article'], 'media' | 'photos'>) {
  const usable = (photos ?? []).filter(photo => safeArticlePhotoUrl(photo.url))
  return media?.url ? { cover: undefined, inline: usable } : { cover: usable[0], inline: usable.slice(1) }
}

export function articleBodyBlocks(body: string) {
  return body.split(/\r?\n/).filter(line => line.trim()).map((text, index) => {
    const heading = text.trim().match(/^#{1,6}[ \t]+(.+?)(?:[ \t]+#+)?$/)?.[1]
    // Older articles put a short title or question at the beginning of a paragraph.
    const prefix = text.trimStart().match(/^([\p{Lu}\p{Lt}][^.!?:]{0,63})([:?])(?=\s|$)/u)
    const title = prefix?.[1].trim()
    const label = heading || (title && title.split(/\s+/).length <= 8
      ? title + (prefix?.[2] === '?' ? '?' : '')
      : undefined)
    const id = label ? `article-section-${index + 1}` : index === 0 ? 'article-text' : undefined
    return { text, heading, label, id }
  })
}
