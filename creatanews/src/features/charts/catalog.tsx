import type {Article,Locale} from '@creata/contracts'
import {graphicCopy,text,validateDataset} from '@creata/graphics'
import {GraphicFigure} from '@creata/graphics/figure'
import {localePath,formatDate} from '@/lib/i18n'
import './charts.css'
import '@creata/graphics/styles.css'

export function ChartsCatalog({articles,locale,type='',query=''}:{articles:Article[];locale:Locale;type?:string;query?:string}){
 const t=graphicCopy[locale],needle=query.trim().toLowerCase()
 const items=articles.filter(a=>a.format==='graphic'&&a.graphic&&!validateDataset(a.graphic)&&a.title[locale]?.trim()&&(!type||a.graphic.type===type)&&(!needle||`${text(a.title,locale)} ${text(a.lead,locale)} ${a.graphic.sourceName}`.toLowerCase().includes(needle))).sort((a,b)=>(b.publishedAt||'').localeCompare(a.publishedAt||''))
 return <div className='cg-catalog'><header className='cg-catalog-head'><p className='cg-eyebrow'>CREATA / DATA</p><h1>{t.title}</h1><p>{t.intro}</p></header>
 <form className='cg-filters' action={localePath(locale,'/charts')}><input name='q' aria-label={locale==='ru'?'Поиск графиков':locale==='kk'?'Графиктерді іздеу':'Search charts'} defaultValue={query} placeholder={locale==='ru'?'Найти тему или источник…':locale==='kk'?'Тақырып немесе дереккөз…':'Find a topic or source…'}/><select name='type' aria-label={locale==='ru'?'Тип графика':locale==='kk'?'График түрі':'Chart type'} defaultValue={type}><option value=''>{t.all}</option>{(['bar','line','area','donut'] as const).map(key=><option value={key} key={key}>{t[key]}</option>)}</select><button type='submit'>{locale==='ru'?'Найти':locale==='kk'?'Іздеу':'Search'}</button></form>
 {items.length?<div className='cg-catalog-grid'>{items.map((a,i)=><article className='cg-story' key={a.id}><a href={localePath(locale,`/charts/${a.slug}`)} aria-label={text(a.title,locale)} className='cg-story-chart'><GraphicFigure data={a.graphic!} locale={locale} title={text(a.title,locale)} compact/></a><div className='cg-story-copy'><div className='cg-story-kicker'><span>{String(i+1).padStart(2,'0')} / {t[a.graphic!.type]}</span><time dateTime={a.publishedAt}>{formatDate(a.publishedAt,locale)}</time></div><h2><a href={localePath(locale,`/charts/${a.slug}`)}>{text(a.title,locale)}</a></h2><p>{text(a.lead,locale)}</p><footer>{a.graphic!.sourceName} · {a.graphic!.period}</footer></div></article>)}</div>:<div className='cg-no-data'><p>{t.empty}</p>{(query||type)&&<a href={localePath(locale,'/charts')}>{t.all}</a>}</div>}
 </div>
}
