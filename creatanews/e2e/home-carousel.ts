import { expect, type APIRequestContext, type Page } from '@playwright/test'

async function preparePopularStories(request: APIRequestContext) {
  const feedSize = 6
  const publicApi = '/api/creata/v1/public'
  const ranked = await request.get(`${publicApi}/popular?locale=ru`)
  await expect(ranked).toBeOK()
  const { items: popular }: { items: { id: string }[] } = await ranked.json()
  const missing = feedSize - popular.length
  if (missing <= 0) return

  const published = await request.get(`${publicApi}/articles?locale=ru`)
  await expect(published).toBeOK()
  const { items }: { items: { id: string }[] } = await published.json()
  const rankedIds = new Set(popular.map(article => article.id))
  const stories = items.filter(article => !rankedIds.has(article.id)).slice(0, missing)
  expect(stories, 'Six published stories are needed to fill the popularity carousel').toHaveLength(missing)
  for (const { id } of stories) {
    const view = await request.post(`${publicApi}/events`, { data: { type: 'view', targetId: id } })
    await expect(view).toBeOK()
  }
}

export async function openHomeCarousel(page: Page, kind: 'spotlight' | 'popular') {
  if (kind === 'popular') await preparePopularStories(page.request)
  const hydrated = page.waitForResponse(response => response.url().endsWith('/auth/me'))
  await page.goto('/ru')
  await hydrated

  const name = kind === 'popular' ? 'Популярное' : 'В центре внимания'
  const scope = page.getByRole('main').getByRole('region', { name, exact: true })
  const viewport = scope.getByRole('group', { name, exact: true })
  await viewport.scrollIntoViewIfNeeded()
  return { scope, viewport }
}
