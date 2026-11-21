import {adminPath} from './demo'
import {applicationUrls} from './demo'
import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import type { List, NewspaperIssue } from '@creata/contracts'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/ui/components/table'
import { useAdmin } from './hooks'
import { Badge, Button, Empty, ErrorPanel, ExternalLink, Field, Loading, Modal, PageHead } from './ui'
import { newspaperText } from './newspaper-copy'
import './newspaper.css'

function CreateIssue({ close }: { close: () => void }) {
  const { api, t, locale } = useAdmin(), client = useQueryClient(), n = (key: Parameters<typeof newspaperText>[0]) => newspaperText(key, locale)
  const [value, setValue] = useState({ number: '', slug: '', title: '', date: '', locale, description: '', items: [] })
  const create = useMutation({ mutationFn: () => api<NewspaperIssue>('/admin/newspapers', { method: 'POST', body: JSON.stringify(value) }), onSuccess: issue => { void client.invalidateQueries({ queryKey: ['newspapers'] }); window.location.assign(adminPath(`/newspapers/${issue.id}`)) } })
  return <Modal title={n('create')} onClose={close}><form className="form-stack" onSubmit={e => { e.preventDefault(); create.mutate() }}>
    <Field label={n('number')}><input required maxLength={40} value={value.number} onChange={e => setValue(v => ({ ...v, number: e.target.value, slug: /^[0-9]+$/.test(e.target.value) ? e.target.value.padStart(2, '0') : v.slug }))}/></Field>
    <Field label={n('slug')}><input required pattern="[a-z0-9]+(-[a-z0-9]+)*" maxLength={80} value={value.slug} onChange={e => setValue(v => ({ ...v, slug: e.target.value }))}/></Field>
    <Field label={n('title')}><input required maxLength={240} value={value.title} onChange={e => setValue(v => ({ ...v, title: e.target.value }))}/></Field>
    <Field label={n('month')}><input required type="month" value={value.date} onChange={e => setValue(v => ({ ...v, date: e.target.value }))}/></Field>
    <Field label={t('language')}><select value={value.locale} onChange={e => setValue(v => ({ ...v, locale: e.target.value as typeof locale }))}><option value="ru">RU</option><option value="kk">ҚАЗ</option><option value="en">EN</option></select></Field>
    {create.error && <ErrorPanel error={create.error}/>}<div className="form-actions"><Button type="button" variant="secondary" onClick={close}>{t('cancel')}</Button><Button disabled={create.isPending}>{n('create')}</Button></div>
  </form></Modal>
}

export function NewspapersPage() {
  const { api, t, locale } = useAdmin(), [creating, setCreating] = useState(false), n = (key: Parameters<typeof newspaperText>[0]) => newspaperText(key, locale)
  const query = useQuery({ queryKey: ['newspapers'], queryFn: () => api<List<NewspaperIssue>>('/admin/newspapers') })
  return <><PageHead title={n('newspapers')} description={n('lead')} action={<><ExternalLink href={`${applicationUrls.web}/${locale}/gazeta`}>{n('archive')}</ExternalLink><Button type="button" onClick={() => setCreating(true)}><Plus size={16}/>{n('create')}</Button></>}/>
    {query.isPending ? <Loading/> : query.error ? <ErrorPanel error={query.error} retry={() => void query.refetch()}/> : !query.data.items.length ? <Empty title={n('empty')}/> : <div className="panel table-panel"><Table><TableHeader><TableRow><TableHead>{n('number')}</TableHead><TableHead>{n('title')}</TableHead><TableHead>{n('month')}</TableHead><TableHead>{t('language')}</TableHead><TableHead>{t('status')}</TableHead><TableHead/></TableRow></TableHeader><TableBody>{query.data.items.map(issue => <TableRow key={issue.id}><TableCell>№{issue.number}</TableCell><TableCell><Link to="/newspapers/$issueId" params={{ issueId: issue.id }}>{issue.title}</Link></TableCell><TableCell>{issue.date}</TableCell><TableCell>{issue.locale.toUpperCase()}</TableCell><TableCell><Badge status={issue.status}/>{issue.publishedRevision && issue.publishedRevision !== issue.revision ? <small className="muted"> · {t('published')}</small> : null}</TableCell><TableCell><Link className="text-link" to="/newspapers/$issueId" params={{ issueId: issue.id }}>{t('edit')}</Link></TableCell></TableRow>)}</TableBody></Table></div>}
    <section className='panel section-spacing'><Link to='/content' search={{page:'newspaper-01'} as never}>{locale==='ru'?'Редактировать оригинальный выпуск №01':locale==='kk'?'Түпнұсқа №01 нөмірді өңдеу':'Edit original issue №01'}</Link></section>
    {creating && <CreateIssue close={() => setCreating(false)}/>}
  </>
}
