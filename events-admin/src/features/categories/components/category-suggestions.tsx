import { CheckIcon, Loader2Icon, PlusIcon } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/shared/ui/components/button'
import { useCategoryCatalog, useCreateCategory } from '../api'
import { categoryPresets, existingCategoryNames } from '../presets'

export function CategorySuggestions() {
  const catalog = useCategoryCatalog()
  const create = useCreateCategory()
  const [pendingName, setPendingName] = useState<string | null>(null)
  const existing = existingCategoryNames(catalog.data?.categories ?? [])

  async function add(preset: (typeof categoryPresets)[number]) {
    if (pendingName || existing.has(preset.name.toLowerCase())) return
    setPendingName(preset.name)
    try {
      await create.mutateAsync({ name: preset.name, color: preset.color })
      await catalog.refetch({ cancelRefetch: false })
      toast.success(`${preset.name} category created`)
    } catch {
      toast.error(`Could not create ${preset.name}. Please try again.`)
    } finally {
      setPendingName(null)
    }
  }

  return (
    <section
      className='rounded-lg border p-4 space-y-3'
      aria-labelledby='suggested-categories-title'
      aria-busy={catalog.isPending || !!pendingName}
    >
      <div>
        <h2 id='suggested-categories-title' className='text-sm font-medium'>
          Suggested categories
        </h2>
        <p className='mt-1 text-sm text-muted-foreground'>
          Add a category to use it when creating and filtering events.
        </p>
      </div>
      {catalog.isPending ? (
        <p className='text-sm text-muted-foreground' role='status'>
          Loading categories…
        </p>
      ) : catalog.isError ? (
        <div className='flex flex-wrap items-center gap-2 text-sm' role='alert'>
          <span>Could not load categories.</span>
          <Button type='button' variant='outline' size='sm' onClick={() => void catalog.refetch()}>
            Try again
          </Button>
        </div>
      ) : (
        <div className='flex flex-wrap gap-2'>
          {categoryPresets.map((preset) => {
            const exists = existing.has(preset.name.toLowerCase())
            return (
              <Button
                type='button'
                key={preset.name}
                variant='outline'
                size='sm'
                disabled={exists || pendingName !== null}
                aria-label={exists ? `${preset.name} category already exists` : `Add ${preset.name} category`}
                onClick={() => void add(preset)}
              >
                <span
                  className='size-2 shrink-0 rounded-full'
                  style={{ backgroundColor: preset.color }}
                  aria-hidden='true'
                />
                {preset.name}
                {pendingName === preset.name ? (
                  <Loader2Icon size={14} className='animate-spin' aria-hidden='true' />
                ) : exists ? (
                  <CheckIcon size={14} aria-hidden='true' />
                ) : (
                  <PlusIcon size={14} aria-hidden='true' />
                )}
              </Button>
            )
          })}
        </div>
      )}
    </section>
  )
}
