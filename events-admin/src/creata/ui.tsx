import { cloneElement, isValidElement, useId, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { AlertCircle, ArrowUpRight, Check, Inbox, LoaderCircle, RefreshCw, X } from 'lucide-react'
import { ApiError } from '@creata/contracts'
import { Button as ExistingButton } from '@/shared/ui/components/button'
import { Input } from '@/shared/ui/components/input'
import { Textarea } from '@/shared/ui/components/textarea'
import { Dialog, DialogContent, DialogTitle } from '@/shared/ui/components/dialog'
import { useAdmin } from './hooks'
import { hasCopy } from './i18n'
import type { Locale } from './types'

export function Button({ variant = 'primary', children, ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary' | 'ghost' | 'danger' }) { return <ExistingButton {...props} type={props.type || 'submit'} variant={variant==='primary'?'default':variant==='secondary'?'outline':variant==='danger'?'destructive':'ghost'} className={`button ${variant} ${props.className || ''}`}>{children}</ExistingButton> }
export function PageHead({ title, description, action }: { title: string; description?: string; action?: ReactNode }) { return <header className='page-head'><div><h1>{title}</h1>{description && <p>{description}</p>}</div>{action && <div className='page-actions'>{action}</div>}</header> }
export function Field({ label, hint, children, className = '' }: { label: string; hint?: string; children: ReactNode; className?: string }) {
  const generated = useId(); const raw = isValidElement<{id?:string;'aria-describedby'?:string}>(children) ? children : null; const control = raw?.type==='input' ? <Input {...raw.props}/> : raw?.type==='textarea' ? <Textarea {...raw.props}/> : raw; const id = control?.props.id || generated
  return <div className={`field ${className}`}><label htmlFor={id}>{label}</label>{control ? cloneElement(control,{id,...(hint?{'aria-describedby':[control.props['aria-describedby'],`${id}-hint`].filter(Boolean).join(' ')}:{})}) : children}{hint&&<small id={`${id}-hint`}>{hint}</small>}</div>
}
export function Empty({ title, description, children }: { title?: string; description?: string; children?: ReactNode }) { const { t } = useAdmin(); return <div className='empty'><span className='empty-icon'><Inbox size={26}/></span><h3>{title || t('emptyTitle')}</h3>{description && <p>{description}</p>}{children}</div> }
export function Loading() { const { t } = useAdmin(); return <div className='loading' role='status'><LoaderCircle size={22} className='spin'/>{t('loading')}</div> }
export function ErrorPanel({ error, retry }: { error: unknown; retry?: () => void }) { const { t } = useAdmin(); const detail = error instanceof ApiError ? error.status === 409 ? t('conflict') : error.status === 403 ? t('forbidden') : error.status===507?error.message:error.status >= 500 ? t('unavailable') : error.message : error instanceof Error ? error.message : t('mutationError'); return <div className='error-panel' role='alert'><AlertCircle size={20}/><div><strong>{t('loadError')}</strong><p>{detail}</p>{retry && <Button variant='secondary' onClick={retry}><RefreshCw size={14}/>{t('retry')}</Button>}</div></div> }
export function Badge({ status }: { status: string }) { const { t } = useAdmin(); return <span className={`badge ${status}`}>{hasCopy(status) ? t(status) : status}</span> }
export function LocaleTabs({ value, onChange }: { value: Locale; onChange: (locale: Locale) => void }) { const { t } = useAdmin(); return <div className='locale-tabs' role='group' aria-label={t('language')}>{(['ru', 'kk', 'en'] as Locale[]).map(l => <button key={l} type='button' aria-pressed={l === value} onClick={() => onChange(l)}>{l === 'kk' ? 'ҚАЗ' : l.toUpperCase()}</button>)}</div> }
export function Modal({ title, children, onClose, wide = false, disablePointerDismissal = false }: { title: string; children: ReactNode; onClose: () => void; wide?: boolean; disablePointerDismissal?:boolean }) {
  const { t } = useAdmin()
  return <Dialog open disablePointerDismissal={disablePointerDismissal} onOpenChange={open=>{if(!open)onClose()}}><DialogContent showCloseButton={false} aria-describedby={undefined} className={`modal ${wide?'wide':''}`}><div className='modal-inner'><header><DialogTitle>{title}</DialogTitle><ExistingButton variant='ghost' size='icon-sm' type='button' onClick={onClose} aria-label={t('close')}><X size={20}/></ExistingButton></header>{children}</div></DialogContent></Dialog>
}
export function ExternalLink({ href, children }: { href: string; children: ReactNode }) { let safe = false; try { safe = ['http:', 'https:'].includes(new URL(href).protocol) } catch {} return safe ? <a href={href} target='_blank' rel='noopener noreferrer' className='external-link'>{children}<ArrowUpRight size={13}/></a> : <span>{children}</span> }
export function SavedIndicator() { const { t } = useAdmin(); return <span className='saved-indicator'><Check size={14}/>{t('saved')}</span> }
