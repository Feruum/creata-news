import Link from 'next/link'
import {ArrowLeft,ArrowUpRight} from 'lucide-react'
import type {Locale,NewspaperPublication} from '@creata/contracts'
import {localePath} from '@/lib/i18n'
import {newspaperCopy,newspaperDate} from './catalog'
import {IssuePhotography} from './issue-photography'
import './newspaper.css'
import './issue-01.css'
export function OriginalNewspaperView({issue,locale}:{issue:NewspaperPublication;locale:Locale}){
  const t = newspaperCopy[locale]
  return <section className="gazeta-reader" aria-label={`${t.issue} №${issue.number}`}>
    <nav className="gazeta-reader-toolbar" aria-label={t.title}>
      <Link href={localePath(locale, '/gazeta')}><ArrowLeft size={16} aria-hidden="true"/>{t.back}</Link>
      <span>{newspaperDate(issue.date, locale)} · {issue.pages} {t.pages} · {t.language}</span>
      <a href={issue.pdf} target="_blank" rel="noopener noreferrer">{t.pdf}<ArrowUpRight size={16} aria-hidden="true"/></a>
    </nav>
    <IssuePhotography><div dangerouslySetInnerHTML={{ __html: issue.html||'' }}/></IssuePhotography>
  </section>
}
