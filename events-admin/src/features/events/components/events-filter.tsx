import { CheckIcon, ChevronRightIcon, FilterIcon, SearchIcon, XIcon } from 'lucide-react'
import { useMemo, useState } from 'react'

import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Checkbox } from '@/shared/ui/components/checkbox'
import { Input } from '@/shared/ui/components/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/components/popover'
import { ScrollArea } from '@/shared/ui/components/scroll-area'
import { cn } from '@/shared/ui/lib/utils'

export type EventsFilterProps = {
  formatFilter: number[]
  statusFilter: string[]
  orgFilter: number[]
  categoryFilter: number[]
  organizationOptions: { value: number; label: string }[]
  categoryOptions: { value: number; label: string; color: string }[]
  statusOptions: { value: string; label: string }[]
  formatOptions: { value: number; label: string }[]
  activeFilterCount: number
  onFormatChange: (checked: boolean, value: number) => void
  onStatusChange: (checked: boolean, value: string) => void
  onOrgChange: (checked: boolean, value: number) => void
  onCategoryChange: (checked: boolean, value: number) => void
}

export function EventsFilter({
  formatFilter,
  statusFilter,
  orgFilter,
  categoryFilter,
  organizationOptions,
  categoryOptions,
  statusOptions,
  formatOptions,
  activeFilterCount,
  onFormatChange,
  onStatusChange,
  onOrgChange,
  onCategoryChange
}: EventsFilterProps) {
  const [orgSubmenuOpen, setOrgSubmenuOpen] = useState(false)
  const [orgSearch, setOrgSearch] = useState('')
  const [categorySubmenuOpen, setCategorySubmenuOpen] = useState(false)
  const [categorySearch, setCategorySearch] = useState('')

  const filteredOrgs = useMemo(() => {
    if (!orgSearch.trim()) return organizationOptions
    const search = orgSearch.toLowerCase()
    return organizationOptions.filter((org) => org.label.toLowerCase().includes(search))
  }, [organizationOptions, orgSearch])

  const filteredCategories = useMemo(() => {
    if (!categorySearch.trim()) return categoryOptions
    const search = categorySearch.toLowerCase()
    return categoryOptions.filter((cat) => cat.label.toLowerCase().includes(search))
  }, [categoryOptions, categorySearch])

  const handleOrgSubmenuChange = (open: boolean) => {
    setOrgSubmenuOpen(open)
    if (!open) setOrgSearch('')
  }

  const handleCategorySubmenuChange = (open: boolean) => {
    setCategorySubmenuOpen(open)
    if (!open) setCategorySearch('')
  }

  return (
    <div className='flex items-center gap-2'>
      <Popover>
        <PopoverTrigger render={<Button variant='outline' size='sm' className='h-9' />}>
          <FilterIcon className='h-4 w-4 mr-2' />
          Filters
          {activeFilterCount > 0 && (
            <span className='ml-2 inline-flex h-5 min-w-5 items-center justify-center rounded-md bg-primary px-1.5 text-xs font-medium text-primary-foreground'>
              {activeFilterCount}
            </span>
          )}
        </PopoverTrigger>
        <PopoverContent align='start' className='w-56 p-0'>
          <div className='p-1'>
            {organizationOptions.length > 0 && (
              <Popover open={orgSubmenuOpen} onOpenChange={handleOrgSubmenuChange}>
                <PopoverTrigger
                  render={
                    <button
                      type='button'
                      className={cn(
                        'flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground',
                        orgFilter.length > 0 && 'font-medium'
                      )}
                    />
                  }
                >
                  <span>Organization</span>
                  <div className='flex items-center gap-1'>
                    {orgFilter.length > 0 && (
                      <Badge variant='secondary' className='h-5 min-w-5 px-1.5 text-xs'>
                        {orgFilter.length}
                      </Badge>
                    )}
                    <ChevronRightIcon className='h-4 w-4 text-muted-foreground' />
                  </div>
                </PopoverTrigger>
                <PopoverContent side='right' align='start' className='w-64 p-0' sideOffset={2}>
                  <div className='p-2 border-b'>
                    <div className='relative'>
                      <SearchIcon className='absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground' />
                      <Input
                        placeholder='Search organizations...'
                        value={orgSearch}
                        onChange={(e) => setOrgSearch(e.target.value)}
                        className='h-8 pl-8 text-sm'
                      />
                    </div>
                  </div>
                  <ScrollArea className='max-h-64'>
                    <div className='p-1'>
                      {filteredOrgs.length === 0 ? (
                        <div className='px-2 py-4 text-center text-sm text-muted-foreground'>
                          No organizations found
                        </div>
                      ) : (
                        filteredOrgs.map((org) => {
                          const isSelected = orgFilter.includes(org.value)
                          return (
                            <button
                              key={org.value}
                              type='button'
                              onClick={() => onOrgChange(!isSelected, org.value)}
                              className='flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground'
                            >
                              <div
                                className={cn(
                                  'flex h-4 w-4 shrink-0 items-center justify-center rounded-sm border',
                                  isSelected
                                    ? 'border-primary bg-primary text-primary-foreground'
                                    : 'border-muted-foreground/30'
                                )}
                              >
                                {isSelected && <CheckIcon className='h-3 w-3' />}
                              </div>
                              <span className='truncate'>{org.label}</span>
                            </button>
                          )
                        })
                      )}
                    </div>
                  </ScrollArea>
                  {orgFilter.length > 0 && (
                    <div className='border-t p-1'>
                      <button
                        type='button'
                        onClick={() => {
                          orgFilter.forEach((id) => {
                            onOrgChange(false, id)
                          })
                        }}
                        className='flex w-full items-center justify-center gap-1 rounded-sm px-2 py-1.5 text-xs text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                      >
                        <XIcon className='h-3 w-3' />
                        Clear all ({orgFilter.length})
                      </button>
                    </div>
                  )}
                </PopoverContent>
              </Popover>
            )}

            {categoryOptions.length > 0 && (
              <Popover open={categorySubmenuOpen} onOpenChange={handleCategorySubmenuChange}>
                <PopoverTrigger
                  render={
                    <button
                      type='button'
                      className={cn(
                        'flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground',
                        categoryFilter.length > 0 && 'font-medium'
                      )}
                    />
                  }
                >
                  <span>Category</span>
                  <div className='flex items-center gap-1'>
                    {categoryFilter.length > 0 && (
                      <Badge variant='secondary' className='h-5 min-w-5 px-1.5 text-xs'>
                        {categoryFilter.length}
                      </Badge>
                    )}
                    <ChevronRightIcon className='h-4 w-4 text-muted-foreground' />
                  </div>
                </PopoverTrigger>
                <PopoverContent side='right' align='start' className='w-64 p-0' sideOffset={2}>
                  <div className='p-2 border-b'>
                    <div className='relative'>
                      <SearchIcon className='absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground' />
                      <Input
                        placeholder='Search categories...'
                        value={categorySearch}
                        onChange={(e) => setCategorySearch(e.target.value)}
                        className='h-8 pl-8 text-sm'
                      />
                    </div>
                  </div>
                  <ScrollArea className='max-h-64'>
                    <div className='p-1'>
                      {filteredCategories.length === 0 ? (
                        <div className='px-2 py-4 text-center text-sm text-muted-foreground'>No categories found</div>
                      ) : (
                        filteredCategories.map((cat) => {
                          const isSelected = categoryFilter.includes(cat.value)
                          return (
                            <button
                              key={cat.value}
                              type='button'
                              onClick={() => onCategoryChange(!isSelected, cat.value)}
                              className='flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent hover:text-accent-foreground'
                            >
                              <div
                                className={cn(
                                  'flex h-4 w-4 shrink-0 items-center justify-center rounded-sm border',
                                  isSelected
                                    ? 'border-primary bg-primary text-primary-foreground'
                                    : 'border-muted-foreground/30'
                                )}
                              >
                                {isSelected && <CheckIcon className='h-3 w-3' />}
                              </div>
                              <span className='h-3 w-3 shrink-0 rounded-full' style={{ backgroundColor: cat.color }} />
                              <span className='truncate'>{cat.label}</span>
                            </button>
                          )
                        })
                      )}
                    </div>
                  </ScrollArea>
                  {categoryFilter.length > 0 && (
                    <div className='border-t p-1'>
                      <button
                        type='button'
                        onClick={() => {
                          categoryFilter.forEach((id) => {
                            onCategoryChange(false, id)
                          })
                        }}
                        className='flex w-full items-center justify-center gap-1 rounded-sm px-2 py-1.5 text-xs text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                      >
                        <XIcon className='h-3 w-3' />
                        Clear all ({categoryFilter.length})
                      </button>
                    </div>
                  )}
                </PopoverContent>
              </Popover>
            )}

            <div className='my-1 border-t' />

            <div className='px-2 py-1'>
              <div className='mb-1 text-xs font-medium text-muted-foreground uppercase tracking-wide'>Status</div>
              <div className='space-y-0.5'>
                {statusOptions.map((option) => (
                  <label
                    key={option.value}
                    htmlFor={`events-status-${option.value}`}
                    className='flex cursor-pointer items-center gap-2 rounded-sm px-1 py-1 hover:bg-accent'
                  >
                    <Checkbox
                      id={`events-status-${option.value}`}
                      checked={statusFilter.includes(option.value)}
                      onCheckedChange={(checked) => onStatusChange(!!checked, option.value)}
                    />
                    <span className='text-sm'>{option.label}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className='my-1 border-t' />

            <div className='px-2 py-1'>
              <div className='mb-1 text-xs font-medium text-muted-foreground uppercase tracking-wide'>Format</div>
              <div className='space-y-0.5'>
                {formatOptions.map((option) => (
                  <label
                    key={option.value}
                    htmlFor={`events-format-${option.value}`}
                    className='flex cursor-pointer items-center gap-2 rounded-sm px-1 py-1 hover:bg-accent'
                  >
                    <Checkbox
                      id={`events-format-${option.value}`}
                      checked={formatFilter.includes(option.value)}
                      onCheckedChange={(checked) => onFormatChange(!!checked, option.value)}
                    />
                    <span className='text-sm'>{option.label}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  )
}
