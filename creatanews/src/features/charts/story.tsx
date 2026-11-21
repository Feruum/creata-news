import Link from 'next/link'
import type {Article,Locale} from '@creata/contracts'
import {GraphicFigure} from '@creata/graphics/figure'
import {graphicCopy,text} from '@creata/graphics'
import {formatDate,localePath} from '@/lib/i18n'
import {SaveButton,ViewEvent} from '@/components/article-actions'
import './charts.css'
import '@creata/graphics/styles.css'
export function GraphicStory({article:a,locale}:{article:Article;locale:Locale}){
 const t=graphicCopy[locale]
 return <article className='cg-catalog cg-detail'><ViewEvent id={a.id}/><Link className='cg-back' href={localePath(locale,'/charts')}>← {t.back}</Link><header className='cg-detail-head'><p className='cg-eyebrow'>CREATA / DATA · {a.graphic!.period}</p><h1>{text(a.title,locale)}</h1><p>{text(a.lead,locale)}</p><div className='cg-byline'><span>{a.author?.name||'Creata News'}</span><time dateTime={a.publishedAt}>{formatDate(a.publishedAt,locale)}</time><SaveButton id={a.id} type='article' locale={locale}/></div></header><GraphicFigure data={a.graphic!} title={text(a.title,locale)} locale={locale}/><div className='cg-story-body'>{text(a.body,locale).split(/\n\n+/).filter(Boolean).map((p,i)=><p key={i}>{p}</p>)}</div>{a.corrections?.length?<details className='cg-corrections'><summary>{locale==='ru'?'История исправлений':locale==='kk'?'Түзетулер тарихы':'Corrections'}</summary>{a.corrections.map((c,i)=><p key={i}>{formatDate(c.publishedAt,locale)} · {c.note}</p>)}</details>:null}</article>
}
