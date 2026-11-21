import Link from 'next/link'
import {ArrowDown,ArrowUpRight} from 'lucide-react'
import {copy,localePath} from '@/lib/i18n'
import {ArticleCard} from '@/components/cards'
import type {Article,Category,Locale,EcosystemContent} from '@creata/contracts'
import {EcosystemGallery,EcosystemNumbers,EcosystemProducts,EcosystemStack} from '@/features/ecosystem/visuals'
import styles from '@/features/ecosystem/ecosystem.module.css'
export function EcosystemView({locale,content,articles,categories,preview=false}:{locale:Locale;content:EcosystemContent;articles:Article[];categories:Category[];preview?:boolean}){
 const t=content.copy[locale]
 return <div className={styles.page} data-testid='ecosystem-page'>
  {preview&&<div className='preview-banner'><strong>{copy[locale].preview}</strong>{copy[locale].previewNote}</div>}
  <section className={styles.hero} id='ecosystem-top' aria-labelledby='ecosystem-title'>
   <div className={styles.heroBackdrop} aria-hidden='true'>CR≡ATA</div>
   <div className={styles.heroTop}><span>{t.eyebrow}</span><span><i aria-hidden='true'/>{content.groupLabel}</span></div>
   <div className={styles.heroGrid}><div className={styles.heroCopy}><h1 id='ecosystem-title'>{t.hero.map(line=><span key={line}>{line}</span>)}</h1><p>{t.intro}</p><div className={styles.heroLinks}><a href='#ecosystem-products' className={styles.heroPrimary}>{t.explore}<ArrowDown size={17} aria-hidden='true'/></a><a href='#ecosystem-who' className={styles.heroSecondary}>{t.meet}<ArrowDown size={14} aria-hidden='true'/></a></div></div><EcosystemGallery locale={locale} content={content}/></div>
   <div className={styles.heroBottom}><span>{t.photoCredit}</span><Link href={localePath(locale,`/gazeta/${encodeURIComponent(content.issueSlug)}`)}>{t.readIssue}<ArrowUpRight size={14} aria-hidden='true'/></Link></div>
   <svg className={styles.wave} viewBox='0 0 1200 70' preserveAspectRatio='none' aria-hidden='true'><path d='M0 30C240 115 350 10 615 25S985 102 1200 22V70H0Z' fill='currentColor'/></svg>
  </section>
  <section className={styles.who} id='ecosystem-who' aria-labelledby='ecosystem-who-title'><div><p className={styles.sectionLabel}>{t.who}</p><h2 id='ecosystem-who-title'>{t.whoTitle}</h2><EcosystemNumbers locale={locale} content={content}/></div><div className={styles.whoBody}><p>{t.whoBody}</p><p>{t.whoSecond}</p><a href={content.groupUrl} className={styles.subtleLink} target='_blank' rel='noopener noreferrer'>{t.group}<ArrowUpRight size={16} aria-hidden='true'/></a></div></section>
  <section className={styles.model} aria-labelledby='ecosystem-model-title'><p className={styles.sectionLabel}>{t.model}</p><div className={styles.sectionHeading}><h2 id='ecosystem-model-title'>{t.modelTitle}</h2><p>{t.modelLead}</p></div><EcosystemStack locale={locale} content={content}/></section>
  <section className={styles.products} id='ecosystem-products' aria-labelledby='ecosystem-products-title'><p className={styles.sectionLabel}>{t.products}</p><div className={styles.sectionHeading}><h2 id='ecosystem-products-title'>{t.productsTitle}</h2><p>{t.productsLead}</p></div><EcosystemProducts locale={locale} content={content}/></section>
  <section className={styles.stories} id='ecosystem-stories' aria-labelledby='ecosystem-stories-title'><p className={styles.sectionLabel}>{t.stories}</p><div className={styles.sectionHeading}><h2 id='ecosystem-stories-title'>{t.storiesTitle}</h2><p>{t.issueBody}</p></div>
   {articles.length>0&&<div className={styles.storiesGrid}>{articles.slice(0,3).map(article=><ArticleCard key={article.id} article={article} locale={locale} categories={categories}/>)}</div>}
   <article className={styles.magazine}><div className={styles.magazineCopy}><span>CR≡ATA NEWS / {content.issueNumber}</span><h3>{t.issueTitle}</h3><p>{t.issueBody}</p><Link href={localePath(locale,`/gazeta/${encodeURIComponent(content.issueSlug)}`)} className={styles.subtleLink}>{t.readIssue}<ArrowUpRight size={17} aria-hidden='true'/></Link></div><Link href={localePath(locale,`/gazeta/${encodeURIComponent(content.issueSlug)}`)} className={styles.magazineCover} aria-label={t.readIssue} tabIndex={-1}><img src={content.issueCover} alt='' loading='lazy' width={300} height={425}/></Link></article>
  </section>
  <section className={styles.closing} aria-labelledby='ecosystem-next-title'><div><p className={styles.sectionLabel}>{t.next}</p><h2 id='ecosystem-next-title'>{t.nextTitle}</h2></div><div><p>{t.nextBody}</p><div className={styles.closingLinks}><Link href={localePath(locale,'/opportunities')} className={styles.closingPrimary}>{t.opportunities}<ArrowUpRight size={16} aria-hidden='true'/></Link><a href={content.contactUrl} className={styles.closingSecondary}>{t.contact}<ArrowUpRight size={16} aria-hidden='true'/></a></div></div></section>
 </div>
}
