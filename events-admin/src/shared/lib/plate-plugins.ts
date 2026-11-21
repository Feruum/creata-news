import {
  BlockquotePlugin,
  BoldPlugin,
  H1Plugin,
  H2Plugin,
  H3Plugin,
  ItalicPlugin,
  StrikethroughPlugin,
  UnderlinePlugin,
} from '@platejs/basic-nodes/react'
import {
  BulletedListPlugin,
  ListItemContentPlugin,
  ListItemPlugin,
  ListPlugin,
  NumberedListPlugin,
} from '@platejs/list-classic/react'
import { createPlatePlugin, ParagraphPlugin } from 'platejs/react'
import { TgEmojiPlugin } from './tg-emoji-plugin'
import { TgEmojiElement } from '@/shared/components/tg-emoji'
import { BlockquoteElement } from '@/shared/ui/components/blockquote-node'
import { H1Element, H2Element, H3Element } from '@/shared/ui/components/heading-node'
import { ParagraphElement } from '@/shared/ui/components/paragraph-node'

const selectAllPlugin = createPlatePlugin({ key: 'selectAllContent' }).overrideEditor(({ editor }) => ({
  transforms: {
    selectAll() {
      // Backspace uses Slate's selection, even before the browser delivers selectionchange.
      editor.tf.select([])
      return true
    }
  }
}))

export function createPlatePlugins() {
  return [
    selectAllPlugin,
    ParagraphPlugin.withComponent(ParagraphElement),
    H1Plugin.withComponent(H1Element),
    H2Plugin.withComponent(H2Element),
    H3Plugin.withComponent(H3Element),
    BlockquotePlugin.withComponent(BlockquoteElement),
    BoldPlugin,
    ItalicPlugin,
    UnderlinePlugin,
    StrikethroughPlugin,
    ListPlugin,
    BulletedListPlugin,
    NumberedListPlugin,
    ListItemPlugin,
    ListItemContentPlugin,
    TgEmojiPlugin.withComponent(TgEmojiElement),
  ]
}
