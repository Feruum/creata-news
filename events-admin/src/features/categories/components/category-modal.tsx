import { zodResolver } from '@hookform/resolvers/zod'
import type { Category } from '@studyverse/proto'
import { Loader2 } from 'lucide-react'
import { useEffect } from 'react'
import { HexColorPicker } from 'react-colorful'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { useCreateCategory, useUpdateCategory } from '@/features/categories'
import { Button } from '@/shared/ui/components/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/shared/ui/components/dialog'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/shared/ui/components/form'
import { Input } from '@/shared/ui/components/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/components/popover'

const PRESET_COLORS = [
  '#3B82F6',
  '#10B981',
  '#8B5CF6',
  '#6B7280',
  '#EF4444',
  '#06B6D4',
  '#F59E0B',
  '#EC4899',
  '#1F2937'
]

const categorySchema = z.object({
  name: z.string().min(1, 'Name is required').max(50, 'Name must be 50 characters or less'),
  color: z.string().regex(/^#[0-9A-Fa-f]{6}$/, 'Invalid hex color')
})

type CategoryFormValues = z.infer<typeof categorySchema>

interface CategoryModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  category?: Category
}

export function CategoryModal({ open, onOpenChange, category }: CategoryModalProps) {
  const isEditing = !!category

  const form = useForm<CategoryFormValues>({
    resolver: zodResolver(categorySchema),
    defaultValues: {
      name: '',
      color: '#3B82F6'
    }
  })

  useEffect(() => {
    if (open) {
      if (category) {
        form.reset({ name: category.name, color: category.color })
      } else {
        form.reset({ name: '', color: '#3B82F6' })
      }
    }
  }, [open, category, form])

  const createMutation = useCreateCategory()
  const updateMutation = useUpdateCategory()

  const isSubmitting = createMutation.isPending || updateMutation.isPending

  const onSubmit = async (values: CategoryFormValues) => {
    try {
      if (isEditing && category) {
        await updateMutation.mutateAsync({
          id: category.id,
          name: values.name,
          color: values.color
        })
        toast.success('Category updated')
      } else {
        await createMutation.mutateAsync({
          name: values.name,
          color: values.color
        })
        toast.success('Category created')
      }
      onOpenChange(false)
    } catch {
      toast.error(isEditing ? 'Failed to update category' : 'Failed to create category')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Category' : 'Add Category'}</DialogTitle>
          <DialogDescription>
            {isEditing ? 'Update the category name and color.' : 'Create a new event category with a color.'}
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className='space-y-6'>
            <FormField
              control={form.control}
              name='name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder='e.g. Academic, Social, Sport...' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='color'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Color</FormLabel>
                  <div className='space-y-3'>
                    <div className='flex items-center gap-3'>
                      <Popover>
                        <PopoverTrigger
                          render={
                            <button
                              type='button'
                              className='size-10 rounded-lg border-2 border-border transition-all hover:scale-105 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2'
                              style={{ backgroundColor: field.value }}
                              aria-label='Pick color'
                            />
                          }
                        ></PopoverTrigger>
                        <PopoverContent className='w-auto p-3' align='start'>
                          <HexColorPicker color={field.value} onChange={field.onChange} />
                        </PopoverContent>
                      </Popover>
                      <FormControl>
                        <Input
                          placeholder='#3B82F6'
                          className='font-mono uppercase w-28'
                          maxLength={7}
                          {...field}
                          onChange={(e) => {
                            let val = e.target.value
                            if (!val.startsWith('#')) val = `#${val}`
                            field.onChange(val.toUpperCase())
                          }}
                        />
                      </FormControl>
                    </div>
                    <div className='flex flex-wrap gap-2'>
                      {PRESET_COLORS.map((color) => (
                        <button
                          key={color}
                          type='button'
                          onClick={() => field.onChange(color)}
                          className='size-7 rounded-full border-2 transition-all hover:scale-110 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2'
                          style={{
                            backgroundColor: color,
                            borderColor: field.value === color ? 'currentColor' : 'transparent'
                          }}
                          aria-label={`Select color ${color}`}
                        />
                      ))}
                    </div>
                  </div>
                  <FormMessage />
                </FormItem>
              )}
            />

            <DialogFooter>
              <Button type='button' variant='outline' onClick={() => onOpenChange(false)} disabled={isSubmitting}>
                Cancel
              </Button>
              <Button type='submit' disabled={isSubmitting}>
                {isSubmitting && <Loader2 className='mr-2 size-4 animate-spin' />}
                {isEditing ? 'Save Changes' : 'Create Category'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
