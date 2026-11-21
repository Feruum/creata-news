import type {Locale,Opportunity} from '@/lib/models'
import {copy,formatDate,localize} from '@/lib/i18n'
import {regionLabel} from '@/lib/topics'
import {OpportunityCard} from './opportunity-card'
import {opportunityType} from './opportunity-copy'
export function OpportunityDetail({locale,item}:{locale:Locale;item:Opportunity}){
 const t=copy[locale]
 return <article className='opportunity-detail'>
    <span className='eyebrow'>{t.opportunities} / {opportunityType(item.type,locale)}</span>
    <h1>{localize(item.title,locale)}</h1>
    <dl className='opportunity-facts'>
      <div><dt>{t.organization}</dt><dd>{item.organization}</dd></div>
      <div><dt>{t.country}</dt><dd>{regionLabel(item.country,locale)}</dd></div>
      <div><dt>{t.deadline}</dt><dd>{formatDate(item.deadline,locale)}</dd></div>
    </dl>
    <div className='opportunity-detail-layout'><p className='opportunity-description'>{localize(item.description,locale)}</p><OpportunityCard opportunity={item} locale={locale}/></div>
  </article>
}
