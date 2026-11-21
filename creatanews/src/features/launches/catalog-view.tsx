import type {Locale,Launch,Category} from '@creata/contracts'
import {AnimatedNumber} from '@/components/skiper'
import {StateMessage} from '@/components/state-message'
import {copy,localePath,localize} from '@/lib/i18n'
import {regionLabel} from '@/lib/topics'
import {launchCopy,launchSocialCopy} from './copy'
import {LaunchCard} from './launch-card'
import {LaunchVotesProvider} from './votes'
import styles from './launches.module.css'
export function LaunchCatalog({locale,data,categories,filters,error=false}:{locale:Locale;data:Launch[];categories:Category[];filters:Record<string,string|undefined>;error?:boolean}){const t=copy[locale],l=launchCopy[locale],s=launchSocialCopy[locale];
  const countries=[...new Set(data.map(item=>item.country).filter(Boolean))].sort()
  const items=data.filter(item=>(!filters.category||item.categoryId===filters.category)&&(!filters.country||item.country===filters.country)&&(!filters.date||item.launchDate===filters.date)&&(!filters.q||`${item.name} ${localize(item.tagline,locale)} ${localize(item.description,locale)} ${(item.tags||[]).join(' ')}`.toLowerCase().includes(filters.q.toLowerCase())))
  if(filters.sort==='popular')items.sort((a,b)=>(b.votes||0)-(a.votes||0)||b.launchDate.localeCompare(a.launchDate)||a.name.localeCompare(b.name))
  return <LaunchVotesProvider locale={locale}>
    <header className={`page-heading ${styles.catalogHeading}`}><h1>{l.title}</h1></header>
    <form className={styles.filters} action={localePath(locale,'/launches')}>
      <label>{t.search}<input name='q' defaultValue={filters.q||''} placeholder={l.query}/></label>
      <label>{t.topics}<select name='category' defaultValue={filters.category||''}><option value=''>{t.allTopics}</option>{categories.map(item=><option key={item.id} value={item.id}>{localize(item.name,locale)}</option>)}</select></label>
      <label>{t.country}<select name='country' defaultValue={filters.country||''}><option value=''>{t.allRegions}</option>{countries.map(country=><option key={country} value={country}>{regionLabel(country,locale)}</option>)}</select></label>
      <label>{s.date}<input name='date' type='date' defaultValue={filters.date||''}/></label>
      <label>{s.sort}<select name='sort' defaultValue={filters.sort}><option value='popular'>{s.popular}</option><option value='newest'>{s.newest}</option></select></label>
      <button className='button button-accent' type='submit'>{t.search}</button>
    </form>
    {error?<StateMessage locale={locale} error/>:items.length?<><p className={styles.count}>{t.results}: <AnimatedNumber value={items.length}/></p><div className={styles.list}>{items.map((launch,index)=><LaunchCard key={launch.id} launch={launch} locale={locale} categories={categories} variant='row' rank={index+1}/>)}</div></>:<StateMessage locale={locale} title={data.length?t.noResults:l.empty} text={data.length?t.searchHint:l.emptyText}/>}
  </LaunchVotesProvider>
}
