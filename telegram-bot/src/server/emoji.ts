import { Elysia, t } from 'elysia'
import { bot } from '../bot'
import { getCachedAssets, setCachedAsset } from '../lib/emoji-cache'
import { env } from '../lib/env'

export const emojiRoutes = new Elysia().post(
  '/emoji/assets',
  async ({ body }) => {
    const { emojiIds } = body
    if (emojiIds.length === 0) {
      return { assets: {} }
    }
    if (emojiIds.length > 200) {
      throw new Error('Maximum 200 emoji IDs per request')
    }

    // Check cache first
    const { cached, uncachedIds } = getCachedAssets(emojiIds)

    if (uncachedIds.length === 0) {
      return { assets: cached }
    }

    // Fetch uncached emoji stickers from Telegram
    const stickers = await bot.api.getCustomEmojiStickers(uncachedIds)

    const assets: Record<string, { url: string; type: string; fallback: string }> = { ...cached }

    for (const sticker of stickers) {
      if (!sticker.custom_emoji_id) continue

      const file = await bot.api.getFile(sticker.file_id)
      if (!file.file_path) continue

      const url = `https://api.telegram.org/file/bot${env.TELEGRAM_BOT_TOKEN}/${file.file_path}`

      let type: 'webp' | 'tgs' | 'webm' = 'webp'
      if (sticker.is_animated) type = 'tgs'
      else if (sticker.is_video) type = 'webm'

      const fallback = sticker.emoji ?? ''

      const asset = setCachedAsset(sticker.custom_emoji_id, { url, type, fallback })
      assets[sticker.custom_emoji_id] = {
        url: asset.url,
        type: asset.type,
        fallback: asset.fallback
      }
    }

    return { assets }
  },
  {
    body: t.Object({
      emojiIds: t.Array(t.String(), { minItems: 0, maxItems: 200 })
    })
  }
)
