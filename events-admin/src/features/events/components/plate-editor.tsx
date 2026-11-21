'use client'

import {
  BoldIcon,
  Heading1Icon,
  Heading2Icon,
  ItalicIcon,
  ListIcon,
  ListOrderedIcon,
  QuoteIcon,
  StrikethroughIcon,
  UnderlineIcon
} from 'lucide-react'
import type { Value } from 'platejs'
import { Plate, usePlateEditor } from 'platejs/react'
import { useEffect, useMemo } from 'react'
import { createPlatePlugins } from '@/shared/lib/plate-plugins'
import { Editor, EditorContainer } from '@/shared/ui/components/editor'
import { MarkToolbarButton } from '@/shared/ui/components/mark-toolbar-button'
import { Toolbar, ToolbarButton, ToolbarGroup } from '@/shared/ui/components/toolbar'
import { TooltipProvider } from '@/shared/ui/components/tooltip'
import { cn } from '@/shared/ui/lib/utils'

interface PlateEditorProps {
  value: Value
  onChange?: (value: Value) => void
  placeholder?: string
  readOnly?: boolean
  className?: string
  minHeight?: string
}

export function PlateEditor({
  value,
  onChange,
  placeholder = 'Enter text...',
  readOnly = false,
  className,
  minHeight = '200px'
}: PlateEditorProps) {
  const plugins = useMemo(() => createPlatePlugins(), [])
  const contentClassName = cn('prose prose-sm dark:prose-invert max-w-none p-3 focus:outline-none', readOnly && 'p-0')

  const editor = usePlateEditor({
    plugins,
    value,
    readOnly
  })

  useEffect(() => {
    const currentValue = JSON.stringify(editor.children)
    const nextValue = JSON.stringify(value)

    if (currentValue !== nextValue) {
      editor.tf.setValue(value)
    }
  }, [editor, value])

  const handleToggleH1 = () => {
    editor.tf.h1.toggle()
  }

  const handleToggleH2 = () => {
    editor.tf.h2.toggle()
  }

  const handleToggleBulletedList = () => {
    editor.tf.ul.toggle()
  }

  const handleToggleNumberedList = () => {
    editor.tf.ol.toggle()
  }

  const handleToggleBlockquote = () => {
    editor.tf.blockquote.toggle()
  }

  if (readOnly) {
    return (
      <Plate editor={editor} readOnly>
        <div className={cn('prose prose-sm dark:prose-invert max-w-none', className)}>
          <EditorContainer>
            <Editor
              variant='none'
              className='p-0 focus:outline-none'
              placeholder={placeholder}
              readOnly
              style={{ minHeight: 'auto' }}
            />
          </EditorContainer>
        </div>
      </Plate>
    )
  }

  return (
    <Plate editor={editor} onChange={({ value: nextValue }) => onChange?.(nextValue)}>
      <TooltipProvider>
        <div className={cn('overflow-hidden rounded-lg border bg-background', className)}>
          <Toolbar className='flex flex-wrap items-center gap-1 border-b bg-muted/30 p-1'>
            <ToolbarGroup>
              <MarkToolbarButton nodeType='bold' aria-label='Bold' tooltip='Bold'>
                <BoldIcon data-icon='inline-start' />
              </MarkToolbarButton>
              <MarkToolbarButton nodeType='italic' aria-label='Italic' tooltip='Italic'>
                <ItalicIcon data-icon='inline-start' />
              </MarkToolbarButton>
              <MarkToolbarButton nodeType='underline' aria-label='Underline' tooltip='Underline'>
                <UnderlineIcon data-icon='inline-start' />
              </MarkToolbarButton>
              <MarkToolbarButton nodeType='strikethrough' aria-label='Strikethrough' tooltip='Strikethrough'>
                <StrikethroughIcon data-icon='inline-start' />
              </MarkToolbarButton>
            </ToolbarGroup>

            <ToolbarGroup>
              <ToolbarButton aria-label='Heading 1' tooltip='Heading 1' onClick={handleToggleH1}>
                <Heading1Icon data-icon='inline-start' />
              </ToolbarButton>
              <ToolbarButton aria-label='Heading 2' tooltip='Heading 2' onClick={handleToggleH2}>
                <Heading2Icon data-icon='inline-start' />
              </ToolbarButton>
            </ToolbarGroup>

            <ToolbarGroup>
              <ToolbarButton aria-label='Bulleted list' tooltip='Bulleted list' onClick={handleToggleBulletedList}>
                <ListIcon data-icon='inline-start' />
              </ToolbarButton>
              <ToolbarButton aria-label='Numbered list' tooltip='Numbered list' onClick={handleToggleNumberedList}>
                <ListOrderedIcon data-icon='inline-start' />
              </ToolbarButton>
              <ToolbarButton aria-label='Blockquote' tooltip='Blockquote' onClick={handleToggleBlockquote}>
                <QuoteIcon data-icon='inline-start' />
              </ToolbarButton>
            </ToolbarGroup>
          </Toolbar>

          <EditorContainer>
            <Editor className={contentClassName} placeholder={placeholder} style={{ minHeight }} />
          </EditorContainer>
        </div>
      </TooltipProvider>
    </Plate>
  )
}
