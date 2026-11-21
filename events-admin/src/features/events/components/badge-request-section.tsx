import { useId } from 'react'
import { useFormContext } from 'react-hook-form'
import { Checkbox } from '@/shared/ui/components/checkbox'
import { FormControl, FormField, FormItem, FormLabel } from '@/shared/ui/components/form'
import type { CreateEventFormData } from '../schema'

export function BadgeRequestSection({ disabled = false }: { disabled?: boolean }) {
  const form = useFormContext<CreateEventFormData>()
  const descriptionId = useId()

  return (
    <fieldset className='space-y-3 rounded-lg border p-4' disabled={disabled}>
      <legend className='px-1 text-sm font-semibold'>Badge requests</legend>
      <p id={descriptionId} className='text-xs text-muted-foreground'>
        Select the badges you want to request. Administrators must approve each badge before participant credit can be
        awarded.
      </p>
      <div className='flex flex-wrap gap-6'>
        {(
          [
            { name: 'socGpaRequested', label: 'SocGpa' },
            { name: 'irosRequested', label: 'iRos' }
          ] as const
        ).map(({ name, label }) => (
          <FormField
            key={name}
            control={form.control}
            name={name}
            render={({ field }) => (
              <FormItem className='flex items-center gap-2'>
                <FormControl>
                  <Checkbox
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={disabled}
                    aria-describedby={descriptionId}
                  />
                </FormControl>
                <FormLabel className='font-normal'>Request {label}</FormLabel>
              </FormItem>
            )}
          />
        ))}
      </div>
    </fieldset>
  )
}
