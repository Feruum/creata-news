import Link from 'next/link'
import { ArrowUpRight } from 'lucide-react'
import type { Locale } from '@/lib/models'
import { localePath } from '@/lib/i18n'
import { newspaperCopy, newspaperDate, type NewspaperIssue } from './catalog'
import { copy } from '@/lib/i18n'
import './newspaper.css'

export function NewspaperArchive({ locale, issues, error = false }: { locale: Locale; issues: NewspaperIssue[]; error?: boolean }) {
  const t = newspaperCopy[locale]
  return <section className="gazeta-archive" aria-labelledby="gazeta-title">
    <header className="gazeta-archive-heading">
      <p>{t.archive}</p>
      <h1 id="gazeta-title">{t.title}<span aria-hidden="true">.</span></h1>
      <p className="gazeta-intro">{t.intro}</p>
    </header>
    {error && <p className="gazeta-load-error" role="alert">{copy[locale].partialResults}. {copy[locale].errorText}</p>}
    <div className="gazeta-issues">
      {issues.map(issue => <article className="gazeta-edition" key={issue.id}>
        <Link className="gazeta-cover" href={localePath(locale, `/gazeta/${issue.id}`)} aria-label={`${t.read} №${issue.number}`}>
          <img src={issue.cover} alt={`Creata News №${issue.number}`} width={1120} height={1752} loading="eager"/>
        </Link>
        <div className="gazeta-edition-copy">
          <p className="gazeta-edition-meta"><time dateTime={issue.date}>{newspaperDate(issue.date, locale)}</time><span>{issue.pages} {t.pages}</span></p>
          <h2><Link href={localePath(locale, `/gazeta/${issue.id}`)}>{issue.title || 'Creata News'} <span>№{issue.number}</span></Link></h2>
          <p className="gazeta-headline" lang={issue.language}>{issue.headline}</p>
          <p className="gazeta-edition-language">{issue.language === 'ru' ? t.language : issue.language === 'kk' ? 'Қазақша' : 'English'}</p>
          <Link className="button button-accent" href={localePath(locale, `/gazeta/${issue.id}`)}>{t.read}<ArrowUpRight size={17} aria-hidden="true"/></Link>
        </div>
      </article>)}
    </div>
  </section>
}
