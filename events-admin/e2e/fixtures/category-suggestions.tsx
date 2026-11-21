import { create } from '@bufbuild/protobuf'
import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { CategoriesService, CategorySchema } from '@studyverse/proto'
import { QueryClientProvider } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from 'sonner'
import { buildCategoryOptions, CategorySuggestions, useCategoryCatalog } from '@/features/categories'
import { queryClient } from '@/shared/lib/api'
import '@/index.css'

const params = new URLSearchParams(location.search)
const categories = [create(CategorySchema, { id: 3, name: 'Career', color: '#FCB900' })]
if (params.has('existingMusic')) categories.push(create(CategorySchema, { id: 70, name: '  mUsIc ', color: '#8B5CF6' }))
const requests: Array<{ name: string; color: string }> = []
let attempted = false
const transport = createRouterTransport(({ service }) => {
  service(CategoriesService, {
    listCategories: () => ({ categories: [...categories], total: categories.length }),
    createCategory: async ({ name, color }) => {
      requests.push({ name, color })
      window.dispatchEvent(new Event('category-requests'))
      await new Promise((resolve) => setTimeout(resolve, 120))
      if (!attempted && params.has('failFirst')) {
        attempted = true
        throw new ConnectError('Synthetic temporary failure', Code.Unavailable)
      }
      const category = create(CategorySchema, { id: 71 + categories.length - 1, name, color })
      categories.push(category)
      return { category }
    }
  })
})

function App() {
  const catalog = useCategoryCatalog()
  const options = buildCategoryOptions(catalog.data?.categories ?? [], categories.slice(0, 1))
  const [recorded, setRecorded] = useState('[]')
  useEffect(() => {
    const update = () => setRecorded(JSON.stringify(requests))
    window.addEventListener('category-requests', update)
    return () => window.removeEventListener('category-requests', update)
  }, [])
  return (
    <main className='max-w-3xl mx-auto p-4 space-y-4'>
      <CategorySuggestions />
      <label>
        Category filter
        <select aria-label='Category filter' className='block border rounded p-2'>
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </label>
      <pre data-testid='category-requests'>{recorded}</pre>
      <Toaster />
    </main>
  )
}

const root = document.getElementById('root')
if (!root) throw new Error('Missing fixture root')
createRoot(root).render(
  <QueryClientProvider client={queryClient}>
    <TransportProvider transport={transport}>
      <App />
    </TransportProvider>
  </QueryClientProvider>
)
