import { cache } from 'react'
import type { Launch, Locale } from '@creata/contracts'
import { buildQuery } from '@/lib/content'
import { loadList, serverApi } from '@/lib/api'

export const getLaunches = (locale:Locale, filters:Record<string,string|undefined>={}) =>
  loadList<Launch>(`/public/launches${buildQuery({...filters,locale})}`)

export const getLaunch = cache((slug:string) => serverApi<Launch>(`/public/launches/${encodeURIComponent(slug)}`))
