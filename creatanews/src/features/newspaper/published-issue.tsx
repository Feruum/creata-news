import Link from 'next/link'
import { ArrowLeft } from 'lucide-react'
import type { NewspaperPublication } from '@creata/contracts'
import type { Locale } from '@/lib/models'
import { copy, formatDate, localePath, localize } from '@/lib/i18n'
import { safeExternalUrl } from '@/lib/content'
import { CreataBrand } from '@creata/brand/logo'
import { newspaperCopy, newspaperDate } from './catalog'
import './newspaper.css'
import './issue-01.css'

export function PublishedNewspaper({ issue, locale, preview = false }: { issue: NewspaperPublication; locale: Locale; preview?: boolean }) {
  const t = newspaperCopy[locale], lang = issue.locale
  return <section className="gazeta-reader" aria-label={`${t.issue} №${issue.number}`}>
    <nav className="gazeta-reader-toolbar" aria-label={t.title}><Link href={localePath(locale, '/gazeta')}><ArrowLeft size={16}/>{t.back}</Link><span>{newspaperDate(issue.date, locale)} · {issue.pages} {t.pages} · {lang === 'ru' ? t.language : lang === 'kk' ? 'Қазақша' : 'English'}</span></nav>
    {preview && <div className="preview-banner"><strong>{copy[locale].preview}</strong>{copy[locale].previewNote}</div>}
    <div className="newspaper-issue" lang={lang}><div className="desk">
      {issue.entries.length === 0 && <section className="sheet"><p>{copy[locale].emptyTitle}</p></section>}
      {issue.entries.map((entry, index) => { const article = entry.article, photo = index === 0 && issue.cover.url ? issue.cover : entry.photo; return <section className={`sheet managed-newspaper-sheet ${index === 0 ? 's1' : ''}`} key={entry.id} aria-label={`${t.issue} №${issue.number} · ${index + 1}`}>
        {index === 0 ? <header className="masthead"><div className="mh-top"><span>№{issue.number}</span><span>{newspaperDate(issue.date, lang)}</span><span>Creata News</span></div><div className="mh-name"><CreataBrand/></div><div className="mh-bottom"><span>{issue.title}</span><span className="mh-pub">Creata Media</span></div></header> : <header className="runner"><span className="rn-rubric">Creata News</span><span>№{issue.number} · {newspaperDate(issue.date, lang)}</span><span className="rn-num">{index + 1}</span></header>}
        {index === 0 && issue.description && <p className="managed-issue-intro">{issue.description}</p>}
        <div className="lead-head"><p className="kicker">{article.author?.name || copy[lang].authorUnknown}{article.publishedAt ? ` · ${formatDate(article.publishedAt, lang)}` : ''}</p>{index === 0 ? <h1 className="lead-h">{localize(article.title, lang)}</h1> : <h2 className="lead-h">{localize(article.title, lang)}</h2>}{localize(article.lead, lang) && <p className="deck">{localize(article.lead, lang)}</p>}</div>
        {photo.url && <figure className="pic managed-newspaper-photo"><div className="frame"><img src={photo.url} alt={photo.caption || localize(article.title, lang)} width={1200} height={800} loading={index === 0 ? 'eager' : 'lazy'}/></div><figcaption>{photo.caption}{photo.credit && ` · ${photo.credit}`}</figcaption></figure>}
        <div className="cols managed-newspaper-body">{localize(article.body, lang).split(/\n\s*\n|\n/).filter(Boolean).map((paragraph, i) => <p key={i}>{paragraph}</p>)}</div>
        {index === 0 && issue.cover.url && entry.photo.url && issue.cover.url !== entry.photo.url && <figure className="pic managed-newspaper-photo"><div className="frame"><img src={entry.photo.url} alt={entry.photo.caption || localize(article.title, lang)} width={1200} height={800} loading="lazy"/></div><figcaption>{entry.photo.caption}{entry.photo.credit && ` · ${entry.photo.credit}`}</figcaption></figure>}
        {localize(article.actions, lang) && <aside className="factbox"><h3 className="rule-h">{copy[lang].action}</h3><p>{localize(article.actions, lang)}</p></aside>}
        {!!article.sources.length && <aside className="managed-newspaper-sources"><h3 className="rule-h">{copy[lang].sources}</h3>{article.sources.filter(source => safeExternalUrl(source.url)).map((source, i) => <a key={i} href={source.url} target="_blank" rel="noopener noreferrer">{source.title || source.url}</a>)}</aside>}
        <footer className="folio"><span>Creata News · №{issue.number} · {index + 1}</span><span>{article.author?.name || 'Creata Media'}</span></footer>
      </section> })}
    </div></div>
  </section>
}
