import { cache } from 'react'
import type { schemas } from '@creata/contracts'
import type { Article, Author, Category, List, Locale, Opportunity } from './models'
import { buildQuery } from './content'
import { demoMode } from '@/demo/config'
export const apiOrigin = () => (process.env.CREATA_API_ORIGIN || 'http://127.0.0.1:5555').replace(/\/$/, '')
export const siteOrigin = () => (process.env.CREATA_PUBLIC_URL || process.env.CREATA_SITE_URL || 'http://localhost:3002').replace(/\/$/, '')
export class PublicApiError extends Error { constructor(public status: number, message: string) { super(message) } }

const catalogResponses = new Map<string, { etag: string; json: string; bytes: number }>()
const catalogByteLimit = 16 * 1024 * 1024
let catalogBytes = 0
function forgetCatalogResponse(key: string) {
  const previous = catalogResponses.get(key)
  if (previous) catalogBytes -= previous.bytes
  catalogResponses.delete(key)
}

const loadServerApi = cache(async (path: string): Promise<unknown> => {
  if(demoMode){try{const{readDemoPublic}=await import('@/demo/api');return readDemoPublic(path)}catch(error){throw new PublicApiError(error&&typeof error==='object'&&'status'in error?Number(error.status):500,error instanceof Error?error.message:'Demo unavailable')}}
  const url = `${apiOrigin()}/api/creata/v1${path}`
  const isCatalog = /^\/public\/(articles(?:\/[^/?]+)?|regions)(?:\?|$)/.test(path)
  const previous = isCatalog ? catalogResponses.get(url) : undefined
  const headers = new Headers()
  if (path.startsWith('/public/') && process.env.CREATA_SSR_TOKEN) {
    headers.set('X-Creata-SSR-Token', process.env.CREATA_SSR_TOKEN)
  }
  if (previous) headers.set('If-None-Match', previous.etag)
  const response = await fetch(url, {headers, cache: 'no-store', redirect: 'error', signal: AbortSignal.timeout(10000)})
  if (previous && response.status === 304) return JSON.parse(previous.json)
  if (previous) forgetCatalogResponse(url)
  if (!response.ok) throw new PublicApiError(response.status, `API returned ${response.status}`)
  const etag = isCatalog ? response.headers.get('ETag') : null
  if (!etag) return response.json()

  const json = await response.text()
  const data: unknown = JSON.parse(json)
  const bytes = Buffer.byteLength(json)
  if (bytes <= 2 * 1024 * 1024) {
    while (catalogBytes + bytes > catalogByteLimit || catalogResponses.size >= 128) {
      const oldest = catalogResponses.keys().next()
      if (oldest.done) break
      forgetCatalogResponse(oldest.value)
    }
    // A concurrent request for this URL may have filled the entry while reading the body.
    forgetCatalogResponse(url)
    catalogResponses.set(url, { etag, json, bytes })
    catalogBytes += bytes
  }
  return data
})

export async function serverApi<T>(path: string): Promise<T> {
  return await loadServerApi(path) as T
}
export type LoadResult<T> = {data: T; error: false} | {data: T; error: true}
export async function loadList<T>(path: string): Promise<LoadResult<T[]>> { try { const result = await serverApi<List<T>>(path); return {data: Array.isArray(result.items) ? result.items : [], error: false} } catch { return {data: [], error: true} } }
export const getCategories = () => loadList<Category>('/public/categories')
export const getAuthors = () => loadList<Author>('/public/authors')
export function getOpportunities(locale: Locale, ids?: string[]): Promise<LoadResult<Opportunity[]>> {
  if (ids?.length === 0) return Promise.resolve({data: [], error: false})
  return loadList<Opportunity>(`/public/opportunities${buildQuery({locale, ids: ids?.join(',')})}`)
}
export const getArticles = (locale: Locale, filters: Record<string,string|undefined> = {}) => loadList<Article>(`/public/articles${buildQuery({...filters,locale})}`)
export async function getArticleCards(locale: Locale, filters: Record<string,string|undefined> = {}): Promise<LoadResult<Article[]>> {
  if (demoMode) return getArticles(locale, filters)
  const result = await loadList<Article | schemas['ArticleSummary']>(
    `/public/articles${buildQuery({...filters, locale, view: 'summary', translated: '1', includeTotal: filters.limit ? '0' : undefined})}`,
  )
  return {...result, data: result.data.map(item => 'body' in item ? item : {...item, body: {ru: '', kk: '', en: ''}})}
}
export const getPopularArticles = (locale: Locale) => loadList<Article>(`/public/popular${buildQuery({locale})}`)
