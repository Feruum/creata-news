interface EmojiAsset {
  url: string
  type: 'webp' | 'tgs' | 'webm'
  fallback: string
  expiresAt: number
}

const cache = new Map<string, EmojiAsset>()

const TTL_MS = 45 * 60 * 1000 // 45 minutes (Telegram URLs expire after ~1 hour)

export function getCachedAsset(emojiId: string): EmojiAsset | null {
  const entry = cache.get(emojiId)
  if (!entry) return null
  if (Date.now() > entry.expiresAt) {
    cache.delete(emojiId)
    return null
  }
  return entry
}

export function setCachedAsset(emojiId: string, asset: Omit<EmojiAsset, 'expiresAt'>): EmojiAsset {
  const entry: EmojiAsset = {
    ...asset,
    expiresAt: Date.now() + TTL_MS
  }
  cache.set(emojiId, entry)
  return entry
}

export function getCachedAssets(emojiIds: string[]): {
  cached: Record<string, EmojiAsset>
  uncachedIds: string[]
} {
  const cached: Record<string, EmojiAsset> = {}
  const uncachedIds: string[] = []

  for (const id of emojiIds) {
    const entry = getCachedAsset(id)
    if (entry) {
      cached[id] = entry
    } else {
      uncachedIds.push(id)
    }
  }

  return { cached, uncachedIds }
}
