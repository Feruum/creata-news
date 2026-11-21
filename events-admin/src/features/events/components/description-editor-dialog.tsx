import { Loader2, XIcon } from 'lucide-react'
import type { Value } from 'platejs'
import { useEffect, useState } from 'react'
import { EMPTY_PLATE_VALUE, isPlateValueEmpty } from '@/shared/lib/plate-utils'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/shared/ui/components/dialog'
import { PlateEditor } from './plate-editor'

interface DescriptionEditorDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialValue: Value
  onSave: (value: Value) => void
  contentClassName?: string
  placeholder?: string
  isSaving?: boolean
  title?: string
  description?: string
  onCancel?: () => void
}

export function DescriptionEditorDialog({
  open,
  onOpenChange,
  initialValue,
  onSave,
  contentClassName,
  placeholder = 'Введите описание мероприятия...',
  isSaving = false,
  title = 'Описание мероприятия',
  description = 'Используйте панель форматирования для редактирования текста',
  onCancel
}: DescriptionEditorDialogProps) {
  const [descriptionDraft, setDescriptionDraft] = useState<Value>(
    initialValue.length > 0 ? initialValue : EMPTY_PLATE_VALUE
  )

  useEffect(() => {
    if (open) {
      setDescriptionDraft(initialValue.length > 0 ? initialValue : EMPTY_PLATE_VALUE)
    }
  }, [initialValue, open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className={
          contentClassName ?? 'flex max-h-[90vh] w-[min(700px,calc(100vw-2rem))] flex-col gap-0 overflow-hidden p-0'
        }
        style={{ overscrollBehavior: 'contain' }}
        showCloseButton={false}
      >
        <div className='flex items-center justify-between border-b px-6 py-4'>
          <div>
            <DialogTitle>{title}</DialogTitle>
            <DialogDescription className='text-xs'>{description}</DialogDescription>
          </div>
          <Button
            type='button'
            variant='ghost'
            size='icon'
            aria-label='Close'
            onClick={() => {
              onCancel?.()
              onOpenChange(false)
            }}
          >
            <XIcon className='h-4 w-4' aria-hidden='true' />
          </Button>
        </div>

        <div className='flex-1 overflow-auto p-6'>
          <PlateEditor
            value={descriptionDraft}
            onChange={setDescriptionDraft}
            placeholder={placeholder}
            minHeight='300px'
          />
        </div>

        <div className='flex items-center justify-end gap-2 border-t px-6 py-4'>
          <Button
            type='button'
            variant='outline'
            onClick={() => {
              onCancel?.()
              onOpenChange(false)
            }}
            disabled={isSaving}
          >
            Отмена
          </Button>
          <Button
            type='button'
            onClick={() => onSave(descriptionDraft)}
            disabled={isSaving || isPlateValueEmpty(descriptionDraft)}
          >
            {isSaving && <Loader2 className='mr-2 h-4 w-4 animate-spin' aria-hidden='true' />}
            {isSaving ? 'Сохранение…' : 'Сохранить'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
