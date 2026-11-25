'use client'
// Actual OppCard from creata-news-v2/source-v2/src/home.jsx. Its image, type
// badge, organisation, audience, deadline, countdown and two-action markup are
// retained. API props replace CN/localStorage; saves persist in the account.
import Link from 'next/link'
import { motion,useReducedMotion } from 'motion/react'
import type { Locale,Opportunity } from '@/lib/models'
import { copy,formatDate,localePath,localize } from '@/lib/i18n'
import { safeExternalUrl } from '@/lib/content'
import { regionLabel } from '@/lib/topics'
import { OpportunityLink,SaveButton } from '@/components/article-actions'
import { Countdown } from './countdown'
import { opportunityCardCopy,opportunityType } from './opportunity-copy'
import styles from './opportunity-theme.module.css'
export function OpportunityCard({opportunity:o,locale:lang,index:i=0}:{opportunity:Opportunity;locale:Locale;index?:number}){
  const t=copy[lang],labels=opportunityCardCopy[lang],reduced=useReducedMotion(),url=safeExternalUrl(o.url)
  const expired=new Date(o.deadline).valueOf()<Date.now(),audience=localize(o.forWhom,lang)
  return <div className='prototype-opportunity-scope' data-opportunity-id={o.id}>
    <motion.article initial={{opacity:0,y:20}} whileInView={{opacity:1,y:0}} viewport={{once:true}} transition={{delay:reduced?0:i*.06}}
      className={`${styles.card} group flex flex-col overflow-hidden rounded-xl`}>
      {o.photo&&<div className='relative aspect-[16/9] overflow-hidden'><img src={o.photo} alt={localize(o.title,lang)} loading='lazy' className='h-full w-full object-cover transition duration-700 group-hover:scale-105'/>
        <span className={`${styles.photoTag} absolute left-3 top-3 rounded-full px-2.5 py-1 font-mono text-[12px]`}>{opportunityType(o.type,lang)}</span></div>}
      <div className='flex flex-1 flex-col p-4'>
        <p className={`${styles.kicker} font-mono text-[12px]`}>{o.organization}{o.country&&` · ${regionLabel(o.country,lang)}`}</p>
        <h3 className='h-title mt-1.5 text-[18px]'><Link href={localePath(lang,`/opportunities/${o.id}`)}>{localize(o.title,lang)}</Link></h3>
        {audience&&<p className={`${styles.muted} mt-1.5 text-[14px]`}>{labels.forWhom}: {audience}</p>}
        <div className='mt-auto pt-4'>
          <p className={`${styles.muted} mb-1.5 font-mono text-[12px]`}>{t.deadline}: <time dateTime={o.deadline}>{formatDate(o.deadline,lang,false,false)}</time></p>
          <Countdown iso={o.deadline} locale={lang}/>
          <div className='mt-4 flex gap-2'>
            {expired?<span className='prototype-expired'>{t.expired}</span>:url&&<OpportunityLink id={o.id} url={url} locale={lang} label={labels.apply} className={`${styles.primary} btn flex-1 text-[13px]`}/>}
            <SaveButton id={o.id} type='opportunity' locale={lang} labels={{save:labels.save,saved:labels.saved}} className={`${styles.secondary} btn text-[13px]`}/>
          </div>
        </div>
      </div>
    </motion.article>
  </div>
}
