import { createPlatePlugin } from 'platejs/react'

export const TgEmojiPlugin = createPlatePlugin({
  key: 'tg_emoji',
  node: {
    isElement: true,
    isInline: true,
    isVoid: true,
    type: 'tg_emoji',
  },
  parsers: {
    html: {
      deserializer: {
        isElement: true,
        parse: ({ element }: { element: HTMLElement }) => ({
          type: 'tg_emoji',
          emojiId: element.getAttribute('emoji-id') ?? '',
          fallback: element.textContent ?? '',
          children: [{ text: '' }],
        }),
        rules: [{ validNodeName: 'TG-EMOJI' }],
      },
    },
  },
})
