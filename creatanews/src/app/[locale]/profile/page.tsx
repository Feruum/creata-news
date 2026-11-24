import { getCategories } from '@/lib/api'
import { isLocale } from '@/lib/i18n'
import { Profile } from '@/components/profile'

export default async function Page({ params, searchParams }: {
  params: Promise<{ locale: string }>
  searchParams: Promise<{ returnTo?: string | string[] }>
}) {
  const [{ locale: raw }, { returnTo }, categories] = await Promise.all([params, searchParams, getCategories()])
  const locale = isLocale(raw) ? raw : 'ru'
  const target = typeof returnTo === 'string' && /^\/(ru|kk|en)(?:\/(?!\/)|$)/.test(returnTo) && !/[\\\x00-\x1f]/.test(returnTo)
    ? returnTo
    : undefined
  return <Profile locale={locale} categories={categories.data} returnTo={target} />
}
