import assert from 'node:assert/strict'
import { test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { useForm } from 'react-hook-form'
import { Button } from './button'
import { Form, FormControl, FormField, FormItem, FormLabel } from './form'
import { Input } from './input'
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from './select'
import { SidebarMenuButton, SidebarProvider } from './sidebar'

test('saved select values render their label before the popup is opened', () => {
  const html = renderToStaticMarkup(
    <Select value='database'>
      <SelectTrigger>
        <SelectValue placeholder='Choose a course' />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          <SelectItem value='database'>Database systems</SelectItem>
        </SelectGroup>
      </SelectContent>
    </Select>
  )
  assert.match(html, /Database systems/)
  assert.doesNotMatch(html, />database</)
})

test('select keeps its placeholder when no value is selected', () => {
  const html = renderToStaticMarkup(
    <Select value={null}>
      <SelectTrigger>
        <SelectValue placeholder='Choose a course' />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value='database'>Database systems</SelectItem>
      </SelectContent>
    </Select>
  )
  assert.match(html, /Choose a course/)
})

test('button render composition produces one anchor with its content', () => {
  const html = renderToStaticMarkup(
    <Button
      nativeButton={false}
      render={(props) => (
        <a href='/schedule' {...props}>
          {props.children}
        </a>
      )}
    >
      Schedule
    </Button>
  )
  assert.match(html, /^<a\b/)
  assert.match(html, /href="\/schedule"/)
  assert.match(html, />Schedule<\/a>$/)
  assert.doesNotMatch(html, /<button/)
})

test('custom sidebar composition preserves anchor semantics', () => {
  const html = renderToStaticMarkup(
    <SidebarProvider>
      <SidebarMenuButton asChild>
        <a href='/events'>Events</a>
      </SidebarMenuButton>
    </SidebarProvider>
  )
  assert.match(html, /<a[^>]*href="\/events"/)
  assert.match(html, />Events<\/a>/)
})

test('form-control composition retains the input value and label association', () => {
  function FormPreview() {
    const form = useForm({ defaultValues: { name: 'Alice' } })
    return (
      <Form {...form}>
        <FormField
          control={form.control}
          name='name'
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input {...field} />
              </FormControl>
            </FormItem>
          )}
        />
      </Form>
    )
  }
  const html = renderToStaticMarkup(<FormPreview />)
  const labelId = html.match(/for="([^"]+)"/)?.[1]
  assert.ok(labelId)
  assert.ok(html.includes(`id="${labelId}"`))
  assert.match(html, /value="Alice"/)
})
