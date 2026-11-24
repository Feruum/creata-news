import { FollowedStories } from '@/components/followed-stories'
import { isLocale } from '@/lib/i18n'
import { storyCopy } from '@/lib/story-copy'

export default async function Page({ params }: { params: Promise<{ locale: string }> }) {
  const { locale: raw } = await params
  const locale = isLocale(raw) ? raw : 'ru'
  return (
    <>
      <div className="page-heading"><h1>{storyCopy[locale].mine}</h1></div>
      <FollowedStories locale={locale} />
    </>
  )
}
