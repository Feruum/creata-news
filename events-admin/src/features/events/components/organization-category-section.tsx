import type { Category, Organization } from '@studyverse/proto'
import { CheckIcon, ChevronsUpDownIcon, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { Avatar, AvatarFallback, AvatarImage } from '@/shared/ui/components/avatar'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/shared/ui/components/command'
import { ImageUpload } from '@/shared/ui/components/file-upload'
import { FormControl, FormField, FormItem, FormMessage } from '@/shared/ui/components/form'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/components/popover'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { cn } from '@/shared/ui/lib/utils'
import type { CreateEventFormData, CreateEventFormInput } from '../schema'
import { SectionHeader } from './section-header'

interface OrganizationCategorySectionProps {
  form: UseFormReturn<CreateEventFormInput, unknown, CreateEventFormData>
  organizations: Organization[]
  categories: Category[]
}

export function OrganizationCategorySection({ form, organizations, categories }: OrganizationCategorySectionProps) {
  const watchOrganizationIds = form.watch('organizationIds')
  const selectedOrganizations = useMemo(() => {
    if (!watchOrganizationIds || watchOrganizationIds.length === 0) return []
    return organizations.filter((org) => watchOrganizationIds.includes(org.id))
  }, [organizations, watchOrganizationIds])

  const [isOrganizationOpen, setIsOrganizationOpen] = useState(false)

  return (
    <div className='space-y-6'>
      <fieldset className='space-y-4 rounded-lg border bg-muted/30 p-4'>
        <legend className='sr-only'>Primary host</legend>
        <SectionHeader title='Primary host' description='Select the organization hosting this event' />

        <FormField
          control={form.control}
          name='organizationIds'
          render={({ field, fieldState }) => (
            <FormItem>
              {selectedOrganizations.length > 0 && (
                <div className='mb-2 flex max-h-20 flex-wrap gap-1.5 overflow-y-auto'>
                  {selectedOrganizations.map((org) => (
                    <Badge key={org.id} variant='secondary' className='gap-1 py-1 text-xs'>
                      <Avatar className='size-4'>
                        <AvatarImage src={org.imageUrl ?? undefined} />
                        <AvatarFallback className='text-[8px]'>{org.title.slice(0, 2).toUpperCase()}</AvatarFallback>
                      </Avatar>
                      <span className='max-w-24 truncate'>{org.title}</span>
                      <button
                        type='button'
                        onClick={() => field.onChange(field.value.filter((id) => id !== org.id))}
                        className='ml-0.5 rounded-full p-0.5 hover:bg-muted-foreground/20'
                        aria-label={`Remove ${org.title}`}
                      >
                        <X className='h-3 w-3' />
                      </button>
                    </Badge>
                  ))}
                </div>
              )}

              <Popover open={isOrganizationOpen} onOpenChange={setIsOrganizationOpen}>
                <PopoverTrigger render={<FormControl />}>
                  <Button
                    variant='outline'
                    role='combobox'
                    aria-expanded={isOrganizationOpen}
                    className={cn(
                      'w-full justify-between',
                      field.value.length === 0 && 'text-muted-foreground',
                      fieldState.invalid && 'border-destructive'
                    )}
                  >
                    <span>
                      {field.value.length === 0
                        ? 'Select primary host…'
                        : selectedOrganizations[0]?.title || 'Select primary host…'}
                    </span>
                    <ChevronsUpDownIcon className='ml-2 h-4 w-4 shrink-0 opacity-50' />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className='w-[--anchor-width] p-0' align='start'>
                  <Command>
                    <CommandInput placeholder='Search…' />
                    <CommandList>
                      <CommandEmpty>No organizations found.</CommandEmpty>
                      <CommandGroup>
                        {organizations.map((org) => {
                          const isSelected = field.value.includes(org.id)
                          const initials = org.title
                            .split(' ')
                            .filter(Boolean)
                            .slice(0, 2)
                            .map((p) => p[0])
                            .join('')
                            .toUpperCase()
                          return (
                            <CommandItem
                              key={org.id}
                              value={`${org.id} ${org.title}`}
                              onSelect={() => {
                                field.onChange([org.id])
                                setIsOrganizationOpen(false)
                              }}
                            >
                              <CheckIcon className={cn('mr-2 h-4 w-4', isSelected ? 'opacity-100' : 'opacity-0')} />
                              <Avatar className='mr-2 size-5'>
                                <AvatarImage src={org.imageUrl ?? undefined} alt={org.title} />
                                <AvatarFallback className='text-[9px]'>{initials || 'ORG'}</AvatarFallback>
                              </Avatar>
                              <span className='truncate'>{org.title}</span>
                            </CommandItem>
                          )
                        })}
                      </CommandGroup>
                    </CommandList>
                  </Command>
                </PopoverContent>
              </Popover>
              <FormMessage />
            </FormItem>
          )}
        />
      </fieldset>

      <fieldset className='space-y-4 rounded-lg border bg-muted/30 p-4'>
        <legend className='sr-only'>Category</legend>
        <SectionHeader title='Category' description='Optional event category' />

        <FormField
          control={form.control}
          name='categoryId'
          render={({ field }) => (
            <FormItem>
              <Select
                onValueChange={(v) => {
                  if (v === null) return
                  return field.onChange(v ? Number(v) : undefined)
                }}
                value={field.value ? String(field.value) : ''}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder='Select a category…'>
                      {field.value && categories.find((c) => c.id === field.value) && (
                        <div className='flex items-center gap-2'>
                          <div
                            className='size-3 rounded-full border'
                            style={{ backgroundColor: categories.find((c) => c.id === field.value)?.color }}
                          />
                          {categories.find((c) => c.id === field.value)?.name}
                        </div>
                      )}
                    </SelectValue>
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {categories.map((category) => (
                    <SelectItem key={category.id} value={String(category.id)}>
                      <div className='flex items-center gap-2'>
                        <div className='size-3 rounded-full border' style={{ backgroundColor: category.color }} />
                        {category.name}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />
      </fieldset>

      <fieldset className='space-y-4 rounded-lg border bg-muted/30 p-4'>
        <legend className='sr-only'>Cover Image</legend>
        <SectionHeader title='Cover Image' description='JPG, PNG, or WebP (max 5 MB)' />

        <FormField
          control={form.control}
          name='coverImage'
          render={({ field }) => (
            <FormItem>
              <FormControl>
                <ImageUpload
                  value={field.value}
                  onChange={field.onChange}
                  aspectRatio='portrait'
                  aspectRatioOptions={['portrait', 'video', 'original']}
                  className='mx-auto w-full max-w-xs'
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </fieldset>
    </div>
  )
}
