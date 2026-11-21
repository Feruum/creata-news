import { ArrowLeft,ArrowRight } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Locale } from '@/lib/models'
import { copy } from '@/lib/i18n'
import styles from './carousel-actions.module.css'

// Restored from the archived Sunday Open CarouselActions; locale is now an
// explicit prop instead of the removed Studyverse browser-profile context.
export function CarouselActions({title,trackId,canScroll,scroll,children,locale}:{title:string;trackId:string;canScroll:{previous:boolean;next:boolean};scroll:(direction:-1|1)=>void;children:ReactNode;locale:Locale}){
  const text=copy[locale]
  return <div className={styles.actions}>{children}{(canScroll.previous||canScroll.next)&&<><button type='button' aria-label={`${text.previous}: ${title}`} aria-controls={trackId} disabled={!canScroll.previous} onClick={()=>scroll(-1)}><ArrowLeft size={18} aria-hidden='true'/></button><button type='button' aria-label={`${text.next}: ${title}`} aria-controls={trackId} disabled={!canScroll.next} onClick={()=>scroll(1)}><ArrowRight size={18} aria-hidden='true'/></button></>}</div>
}
