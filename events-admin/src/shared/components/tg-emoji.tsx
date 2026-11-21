import { useEffect, useState } from 'react'
import type { PlateElementProps } from 'platejs/react'
import { PlateElement } from 'platejs/react'

interface TgEmojiNodeElement {
  [key: string]: unknown
  type: 'tg_emoji'
  emojiId: string
  fallback: string
  children: [{ text: '' }]
}

interface TgEmojiProps {
  emojiId: string
  fallback: string
  size?: number
}

interface EmojiAsset {
  url: string
  type: 'webp' | 'tgs' | 'webm'
  fallback: string
}

// Simple in-memory batch cache
const assetCache = new Map<string, EmojiAsset>()
const pendingIds = new Set<string>()
let batchTimer: ReturnType<typeof setTimeout> | null = null
let batchCallbacks: Array<() => void> = []

// The emoji endpoint URL — adjust if telegram bot runs on a different host
const EMOJI_API_URL = '/api/emoji/assets'

async function fetchBatch() {
  const ids = Array.from(pendingIds)
  pendingIds.clear()
  const callbacks = batchCallbacks
  batchCallbacks = []
  batchTimer = null

  if (ids.length === 0) return

  try {
    const res = await fetch(EMOJI_API_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ emojiIds: ids }),
    })

    if (res.ok) {
      const data = await res.json() as { assets: Record<string, EmojiAsset> }
      for (const [id, asset] of Object.entries(data.assets)) {
        assetCache.set(id, asset)
      }
    }
  } catch {
    // Silently fail — fallback emoji will show
  }

  for (const cb of callbacks) cb()
}

function requestAsset(emojiId: string, onReady: () => void) {
  if (assetCache.has(emojiId)) {
    onReady()
    return
  }
  pendingIds.add(emojiId)
  batchCallbacks.push(onReady)
  if (!batchTimer) {
    batchTimer = setTimeout(fetchBatch, 50) // 50ms batching window
  }
}

function TgEmoji({ emojiId, fallback, size = 20 }: TgEmojiProps) {
  const [asset, setAsset] = useState<EmojiAsset | null>(
    assetCache.get(emojiId) ?? null,
  )

  useEffect(() => {
    if (asset) return
    requestAsset(emojiId, () => {
      setAsset(assetCache.get(emojiId) ?? null)
    })
  }, [emojiId, asset])

  if (!asset) {
    return <span className="inline" title={`Emoji ${emojiId}`}>{fallback}</span>
  }

  if (asset.type === 'webp') {
    return (
      <img
        src={asset.url}
        width={size}
        height={size}
        alt={asset.fallback}
        className="inline align-text-bottom"
        loading="lazy"
      />
    )
  }

  if (asset.type === 'webm') {
    return (
      <video
        src={asset.url}
        width={size}
        height={size}
        autoPlay
        loop
        muted
        playsInline
        className="inline align-text-bottom"
      />
    )
  }

  if (asset.type === 'tgs') {
    return <TgsPlayer url={asset.url} size={size} fallback={asset.fallback} />
  }

  return <span>{fallback}</span>
}

// Lazy-loaded TGS (Lottie) player to avoid bundling lottie-react for non-animated emoji
function TgsPlayer({ url, size, fallback }: { url: string; size: number; fallback: string }) {
  const [animationData, setAnimationData] = useState<unknown>(null)

  useEffect(() => {
    let cancelled = false

    async function load() {
      try {
        const { decompressSync } = await import('fflate')
        const res = await fetch(url)
        const arrayBuffer = await res.arrayBuffer()
        const decompressed = decompressSync(new Uint8Array(arrayBuffer))
        const json = JSON.parse(new TextDecoder().decode(decompressed))
        if (!cancelled) setAnimationData(json)
      } catch {
        // Fallback handled by parent
      }
    }

    load()
    return () => { cancelled = true }
  }, [url])

  if (!animationData) {
    return <span className="inline">{fallback}</span>
  }

  return <LottieWrapper animationData={animationData} size={size} />
}

function LottieWrapper({ animationData, size }: { animationData: unknown; size: number }) {
  const [Lottie, setLottie] = useState<React.ComponentType<{
    animationData: unknown
    loop: boolean
    autoplay: boolean
    style: React.CSSProperties
  }> | null>(null)

  useEffect(() => {
    import('lottie-react').then((mod) => {
      setLottie(() => mod.default)
    })
  }, [])

  if (!Lottie) return null

  return (
    <Lottie
      animationData={animationData}
      loop
      autoplay
      style={{
        width: size,
        height: size,
        display: 'inline-block',
        verticalAlign: 'text-bottom',
      }}
    />
  )
}

// Plate element component for TgEmojiPlugin
export function TgEmojiElement(props: PlateElementProps<TgEmojiNodeElement>) {
  const { element, children } = props

  return (
    <PlateElement {...props}>
      <span contentEditable={false}>
        <TgEmoji emojiId={element.emojiId} fallback={element.fallback} />
      </span>
      {children}
    </PlateElement>
  )
}
