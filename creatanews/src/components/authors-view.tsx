import Link from 'next/link'
import type {Author,Locale} from '@/lib/models'
import {copy,localePath} from '@/lib/i18n'
import {StateMessage} from '@/components/state-message'
export function AuthorsView({locale,authors,error=false}:{locale:Locale;authors:Author[];error?:boolean}){const t=copy[locale];return <section className='listing-section'><div className='page-heading'><span className='eyebrow'>Creata News</span><h1>{t.authors}</h1></div>{error?<StateMessage locale={locale} error/>:authors.length?<div className='author-grid'>{authors.map(author=><Link href={localePath(locale,`/authors/${author.id}`)} key={author.id} className='author-card'><div className='author-avatar'>{author.avatarUrl?<img src={author.avatarUrl} alt='' loading='lazy'/>:author.name.charAt(0)}</div><div><h2>{author.name}</h2><p>{author.role||t.authorUnknown}</p></div></Link>)}</div>:<StateMessage locale={locale}/>}</section>}
