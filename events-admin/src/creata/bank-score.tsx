import { useId, useState, type ReactNode } from 'react'
import { Popover } from '@base-ui/react/popover'
import { Info } from 'lucide-react'
import { useAdmin } from './hooks'
import './bank-score.css'

function BankHint({label,trigger,heading,children,className}:{label:string;trigger:ReactNode;heading:string;children:ReactNode;className:string}) {
 const [open,setOpen]=useState(false)
 const id=useId()
 return <Popover.Root open={open} onOpenChange={setOpen} triggerId={id}>
  <Popover.Trigger id={id} type='button' className={className} aria-label={label} openOnHover delay={120} closeDelay={120} onFocus={event=>{if(event.currentTarget.matches(':focus-visible'))setOpen(true)}}>{trigger}</Popover.Trigger>
  <Popover.Portal><Popover.Positioner side='top' align='center' sideOffset={8} collisionPadding={12} className='bank-hint-positioner'>
   <Popover.Popup initialFocus={false} finalFocus={false} className='bank-hint-popup'>
    <Popover.Title className='bank-hint-title'>{heading}</Popover.Title>
    <Popover.Description className='bank-hint-description'>{children}</Popover.Description>
   </Popover.Popup>
  </Popover.Positioner></Popover.Portal>
 </Popover.Root>
}

export function HermesScoreHeading(){
 const {t}=useAdmin()
 return <span className='bank-score-heading'>{t('scoreHermes')}<BankHint label={t('threshold')} trigger={<Info size={16} aria-hidden='true'/>} heading={t('threshold')} className='bank-score-help'>{t('scoreThresholdHint')}</BankHint></span>
}

export function HermesScore({score,reason}:{score:number|undefined;reason?:string}){
 const {t}=useAdmin()
 const value=Number(score||0).toFixed(0)
 return <BankHint label={`${t('reason')}: ${value}`} trigger={value} heading={t('reason')} className='score-pill bank-score-trigger'>{reason?.trim()||t('scoreReasonEmpty')}</BankHint>
}
