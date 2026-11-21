'use client'
// Actual OppsBand header and card structure from the prototype. The user's
// carousel request extends its static grid; colors follow the current theme.
import Link from 'next/link'
import type { Locale,Opportunity } from '@/lib/models'
import { localePath,copy } from '@/lib/i18n'
import { StateMessage } from '@/components/state-message'
import { OpportunityCarousel } from './opportunity-carousel'
import { useLiveOpportunities } from './use-live-opportunities'
import styles from './opportunity-theme.module.css'
const labels={ru:{title:'Возможности с дедлайном',lead:'Гранты, конкурсы и стажировки, которые проверил модератор.'},kk:{title:'Мерзімі бар мүмкіндіктер',lead:'Редактор тексерген гранттар, байқаулар және тағылымдамалар.'},en:{title:'Opportunities with deadlines',lead:'Grants, contests and internships checked by a moderator.'}}
export function OpportunitiesBand({opportunities,locale,error=false}:{opportunities:Opportunity[];locale:Locale;error?:boolean}){
  const t=labels[locale],live=useLiveOpportunities(opportunities,locale,error),items=[...live.items].sort((a,b)=>new Date(a.deadline).valueOf()-new Date(b.deadline).valueOf())
  return <div className='prototype-opportunity-scope prototype-opportunities-band' data-testid='opportunities-band'>
    <section className={`${styles.band} py-14`}>
      <div className='wrap'>
        <div className={`${styles.heading} mb-6 flex items-end justify-between border-t-2 pt-3`}>
          <div><h2 className='h-title text-[26px] sm:text-[32px]'><Link href={localePath(locale,'/opportunities')} className='hover:text-[#8C87FF]'>{t.title} →</Link></h2><p className='mt-1 font-serif opacity-70'>{t.lead}</p></div>
        </div>
        {live.error&&!items.length?<StateMessage locale={locale} error/>:items.length?<OpportunityCarousel items={items} locale={locale} title={t.title}/>:<p className='original-empty'>{copy[locale].opportunityEmpty}</p>}
        {live.error&&items.length>0&&<p className='opportunity-refresh-error' role='status'>{copy[locale].operationFailed} <button type='button' onClick={()=>void live.refresh()}>{copy[locale].retry}</button></p>}
      </div>
    </section>
  </div>
}
