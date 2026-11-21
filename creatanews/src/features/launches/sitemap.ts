import type { MetadataRoute } from 'next'
import { locales } from '@/lib/i18n'
import { getLaunches } from './data'

export async function launchSitemap(origin:string):Promise<MetadataRoute.Sitemap> {
  const entries=await Promise.all(locales.map(async locale=>{
    const launches=await getLaunches(locale)
    return [
      {url:`${origin}/${locale}/launches`,changeFrequency:'daily' as const},
      ...launches.data.filter(item=>item.status==='published').map(item=>({url:`${origin}/${locale}/launches/${item.slug}`,lastModified:item.updatedAt,alternates:{languages:Object.fromEntries(locales.map(lang=>[lang,`${origin}/${lang}/launches/${item.slug}`]))}})),
    ]
  }))
  return entries.flat()
}
