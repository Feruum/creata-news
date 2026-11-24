import type { MetadataRoute } from 'next'
import { getArticleCards, getAuthors, getCategories, getOpportunities, siteOrigin } from '@/lib/api'
import { locales } from '@/lib/i18n'
import { publicArticles } from '@/lib/content'
import { getNewspaperArchive } from '@/features/newspaper/public-api'
import { launchSitemap } from '@/features/launches/sitemap'

export const dynamic = 'force-dynamic'

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const origin = siteOrigin()
  const [newspaperArchive, categories, authors] = await Promise.all([
    getNewspaperArchive(), getCategories(), getAuthors(),
  ])
  const result: MetadataRoute.Sitemap = []

  for (const locale of locales) {
    for (const page of ['', '/charts', '/latest', '/ecosystem', '/opportunities', '/authors', '/about']) {
      result.push({ url: `${origin}/${locale}${page}`, changeFrequency: page === '' ? 'daily' : 'weekly' })
    }
    for (const category of categories.data) {
      result.push({ url: `${origin}/${locale}/categories/${category.slug || category.id}` })
    }
    for (const author of authors.data) {
      result.push({ url: `${origin}/${locale}/authors/${author.id}` })
    }

    const [articles, opportunities] = await Promise.all([getArticleCards(locale), getOpportunities(locale)])
    for (const article of publicArticles(articles.data, locale)) {
      const section = article.format === 'graphic' ? 'charts' : 'articles'
      const path = `/${section}/${encodeURIComponent(article.slug)}`
      result.push({
        url: `${origin}/${locale}${path}`,
        lastModified: article.updatedAt,
        alternates: {
          languages: Object.fromEntries(locales.filter(lang => article.title[lang]?.trim()).map(lang => [
            lang, `${origin}/${lang}${path}`,
          ])),
        },
      })
    }
    for (const opportunity of opportunities.data.filter(item => item.status === 'approved')) {
      result.push({ url: `${origin}/${locale}/opportunities/${opportunity.id}` })
    }
  }

  return result.concat(await launchSitemap(origin), locales.flatMap(locale => [
    { url: `${origin}/${locale}/gazeta`, changeFrequency: 'monthly' as const },
    ...newspaperArchive.data.map(issue => ({
      url: `${origin}/${locale}/gazeta/${issue.id}`,
      changeFrequency: 'never' as const,
    })),
  ]))
}
