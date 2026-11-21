import type { Article, Author, Category, List, Locale, Opportunity } from './models'
import { buildQuery } from './content'
import { demoMode } from '@/demo/config'
export const apiOrigin = () => (process.env.CREATA_API_ORIGIN || 'http://127.0.0.1:5555').replace(/\/$/, '')
export const siteOrigin = () => (process.env.CREATA_PUBLIC_URL || process.env.CREATA_SITE_URL || 'http://localhost:3002').replace(/\/$/, '')
export class PublicApiError extends Error { constructor(public status: number, message: string) { super(message) } }
export async function serverApi<T>(path: string): Promise<T> {
  if(demoMode){try{const{readDemoPublic}=await import('@/demo/api');return readDemoPublic(path) as T}catch(error){throw new PublicApiError(error&&typeof error==='object'&&'status'in error?Number(error.status):500,error instanceof Error?error.message:'Demo unavailable')}}
  const response = await fetch(`${apiOrigin()}/api/creata/v1${path}`, {cache: 'no-store', signal: AbortSignal.timeout(10000)})
  if (!response.ok) throw new PublicApiError(response.status, `API returned ${response.status}`)
  return response.json() as Promise<T>
}
export type LoadResult<T> = {data: T; error: false} | {data: T; error: true}
export async function loadList<T>(path: string): Promise<LoadResult<T[]>> { try { const result = await serverApi<List<T>>(path); return {data: Array.isArray(result.items) ? result.items : [], error: false} } catch { return {data: [], error: true} } }
export const getCategories = () => loadList<Category>('/public/categories')
export const getAuthors = () => loadList<Author>('/public/authors')
export const getOpportunities = (locale: Locale) => loadList<Opportunity>(`/public/opportunities${buildQuery({locale})}`)
export const getArticles = (locale: Locale, filters: Record<string,string|undefined> = {}) => loadList<Article>(`/public/articles${buildQuery({...filters,locale})}`)
export const getPopularArticles = (locale: Locale) => loadList<Article>(`/public/popular${buildQuery({locale})}`)
