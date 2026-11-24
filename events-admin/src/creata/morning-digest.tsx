import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import type { MorningDigestStatus } from '@creata/contracts'
import { useAdmin } from './hooks'
import { formatDate } from './i18n'
import { Badge, ErrorPanel, Loading } from './ui'

export function MorningDigestSummary() {
 const { api, t, locale } = useAdmin()
 const query = useQuery({
  queryKey: ['morningDigest'],
  queryFn: () => api<MorningDigestStatus>('/admin/digests/morning'),
  refetchInterval: 30_000,
 })
 const edition = query.data?.edition
 const collectedAt = query.data?.lastCollectionAt
 return <section className='panel mb-5'>
  <div className='section-head'>
   <h2>{t('morningDigest')}</h2>
   {edition && <Link to='/digests' search={{ edition: edition.id }} className='text-button'>{t('openMorningDigest')}</Link>}
  </div>
  {query.isPending ? <Loading/> : query.error ? <ErrorPanel error={query.error} retry={() => void query.refetch()}/> : <>
   {edition ? (
    <div className='flex flex-wrap items-center gap-2 text-sm'>
     <time dateTime={edition.date}>{formatDate(edition.date, locale, false, true)}</time>
     <Badge status={edition.status}/>
     {edition.articleCount + edition.opportunityCount === 0
      ? <span>{t('morningDigestEmpty')}</span>
      : <span>{t('articles')} · {edition.articleCount} / {t('opportunities')} · {edition.opportunityCount}</span>}
    </div>
   ) : <p className='muted text-sm'>{t('morningDigestPending')}</p>}
   <p className='muted mt-3 text-sm'>
    {t('lastHermesCollection')} · {collectedAt
     ? <time dateTime={collectedAt}>{formatDate(collectedAt, locale, true, true)}</time>
     : t('lastHermesCollectionMissing')}
   </p>
  </>}
 </section>
}
