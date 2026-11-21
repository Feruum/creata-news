import {applicationUrls} from './demo'
import { useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowDown, ArrowLeft, ArrowUp, ImagePlus, Plus, Save, Trash2 } from 'lucide-react'
import type { Article, List, Media, NewspaperIssue, NewspaperItem } from '@creata/contracts'
import { useAdmin } from './hooks'
import { localized } from './domain'
import { Badge, Button, ErrorPanel, ExternalLink, Field, Loading, Modal, PageHead } from './ui'
import { newspaperText } from './newspaper-copy'
import { NewspaperPhoto } from './newspaper-photo'
import './newspaper.css'

const emptyPhoto = (): Media => ({ url: '', caption: '', credit: '', rights: '' })
export function NewspaperEditor() {
  const { issueId } = useParams({ strict: false }) as { issueId: string }, { api } = useAdmin()
  const query = useQuery({ queryKey: ['newspaper', issueId], queryFn: () => api<NewspaperIssue>(`/admin/newspapers/${encodeURIComponent(issueId)}`) })
  if (query.isPending) return <Loading/>
  if (query.error) return <ErrorPanel error={query.error} retry={() => void query.refetch()}/>
  return <IssueForm key={`${query.data.id}:${query.data.revision}:${query.data.status}`} initial={query.data} reload={() => void query.refetch()}/>
}

function IssueForm({ initial, reload }: { initial: NewspaperIssue; reload: () => void }) {
  const { api, t, locale, session, toast } = useAdmin(), client = useQueryClient(), n = (key: Parameters<typeof newspaperText>[0]) => newspaperText(key, locale)
  const [draft, setDraft] = useState(() => structuredClone(initial)), [search, setSearch] = useState(''), [photoId, setPhotoId] = useState<string | null>(null), [action, setAction] = useState<'publish' | 'unpublish' | null>(null), [uploads, setUploads] = useState<Record<string, boolean>>({})
  const articles = useQuery({ queryKey: ['newspaper-articles', draft.locale], queryFn: () => api<List<Article>>(`/public/articles?locale=${draft.locale}`) })
  const dirty = JSON.stringify(initial) !== JSON.stringify(draft), uploading = Object.values(uploads).some(Boolean), canEdit = !!session?.user?.roles.some(role => ['editor','chief_editor','admin'].includes(role)) || session?.user?.id === initial.creatorId, canPublish = !!session?.user?.roles.some(role => ['chief_editor','admin'].includes(role))
  const update = <K extends keyof NewspaperIssue>(key: K, value: NewspaperIssue[K]) => setDraft(v => ({ ...v, [key]: value }))
  const refresh = (issue: NewspaperIssue) => { setDraft(structuredClone(issue)); client.setQueryData(['newspaper', issue.id], issue); void client.invalidateQueries({ queryKey: ['newspapers'] }); setAction(null); toast(t('saved')) }
  const save = useMutation({ mutationFn: () => api<NewspaperIssue>(`/admin/newspapers/${initial.id}`, { method: 'PATCH', body: JSON.stringify({ ...draft, expectedRevision: initial.revision }) }), onSuccess: refresh })
  const publish = useMutation({ mutationFn: () => api<NewspaperIssue>(`/admin/newspapers/${initial.id}/${action}`, { method: 'POST', body: JSON.stringify({ expectedRevision: initial.revision }) }), onSuccess: refresh })
  const pending = uploading || save.isPending || publish.isPending, list = articles.data?.items || [], selected = draft.items, chosen = selected.find(item => item.id === photoId)
  const changeItem = (id: string, photo?: Media) => update('items', selected.map(item => item.id === id ? { ...item, photo } : item))
  const move = (index: number, offset: number) => { const items = [...selected]; [items[index], items[index + offset]] = [items[index + offset], items[index]]; update('items', items) }
  const source = (item: NewspaperItem) => list.find(article => article.id === item.articleId)
  const web = applicationUrls.web
  return <><Link className="back-link" to="/newspapers"><ArrowLeft size={15}/>{n('back')}</Link><PageHead title={`${n('newspapers')} №${initial.number}`} description={initial.title} action={<Badge status={initial.status}/>}/>
    {!!initial.publishedRevision && initial.publishedRevision !== initial.revision && <p className="callout">{n('changed')}</p>}
    <div className="newspaper-workspace">
      <section className="panel form-stack"><h2>{n('title')}</h2>
        <Field label={n('title')}><input maxLength={240} disabled={!canEdit || pending} value={draft.title} onChange={e => update('title', e.target.value)}/></Field>
        <div className="form-grid"><Field label={n('number')}><input disabled={!canEdit || pending} value={draft.number} onChange={e => update('number', e.target.value)}/></Field><Field label={n('month')}><input type="month" disabled={!canEdit || pending} value={draft.date} onChange={e => update('date', e.target.value)}/></Field></div>
        <Field label={t('language')}><select disabled={!canEdit || pending} value={draft.locale} onChange={e => update('locale', e.target.value as typeof locale)}><option value="ru">RU</option><option value="kk">ҚАЗ</option><option value="en">EN</option></select></Field>
        <Field label={n('intro')}><textarea rows={4} maxLength={8000} disabled={!canEdit || pending} value={draft.description} onChange={e => update('description', e.target.value)}/></Field>
        <h2>{n('cover')}</h2><NewspaperPhoto value={draft.cover} onChange={photo => update('cover', photo)} onBusy={busy => setUploads(v => ({ ...v, cover: busy }))} disabled={!canEdit || save.isPending || publish.isPending}/>
      </section>
      <section className="panel form-stack"><h2>{n('articles')} <span className="muted">{selected.length}</span></h2><p className="muted">{n('noArticles')}</p>
        <ol className="newspaper-selected">{selected.map((item, index) => { const article = source(item); return <li key={item.id}>
          <span className="newspaper-page-number">{index + 1}</span><div className="newspaper-selected-text"><strong>{article ? localized(article.title, draft.locale) : n('missingArticle')}</strong><small>{item.photo?.url ? item.photo.caption || n('photo') : n('defaultPhoto')}</small></div>
          <div className="newspaper-page-actions"><Button type="button" variant="ghost" aria-label={`${n('up')} ${index + 1}`} disabled={!canEdit || pending || index === 0} onClick={() => move(index, -1)}><ArrowUp size={15}/></Button><Button type="button" variant="ghost" aria-label={`${n('down')} ${index + 1}`} disabled={!canEdit || pending || index === selected.length - 1} onClick={() => move(index, 1)}><ArrowDown size={15}/></Button><Button type="button" variant="ghost" aria-label={`${n('photo')} ${index + 1}`} disabled={!canEdit || pending} onClick={() => setPhotoId(item.id)}><ImagePlus size={15}/></Button><Button type="button" variant="ghost" aria-label={`${t('remove')} ${index + 1}`} disabled={!canEdit || pending} onClick={() => update('items', selected.filter(value => value.id !== item.id))}><Trash2 size={15}/></Button></div>
        </li> })}</ol>
        <h2>{n('available')}</h2><Field label={n('search')}><input type="search" value={search} onChange={e => setSearch(e.target.value)}/></Field>
        {articles.isPending ? <Loading/> : articles.error ? <ErrorPanel error={articles.error} retry={() => void articles.refetch()}/> : <div className="newspaper-picker">{list.filter(article => localized(article.title, draft.locale).toLocaleLowerCase().includes(search.toLocaleLowerCase()) && !!article.title[draft.locale] && !!article.body[draft.locale]).map(article => { const included = selected.some(item => item.articleId === article.id); return <div className="newspaper-picker-item" key={article.id}><span>{localized(article.title, draft.locale)}</span><Button type="button" variant="secondary" disabled={!canEdit || pending || included || selected.length >= 80} onClick={() => update('items', [...selected, { id: crypto.randomUUID(), articleId: article.id }])}><Plus size={14}/>{n(included ? 'selected' : 'add')}</Button></div> })}</div>}
      </section>
    </div>
    {(save.error || publish.error) && <ErrorPanel error={save.error || publish.error} retry={reload}/>}
    <footer className="editor-controls newspaper-controls"><div><Button type="button" disabled={!canEdit || !dirty || pending || !draft.title.trim()} onClick={() => save.mutate()}><Save size={15}/>{save.isPending ? t('saving') : t('save')}</Button>{dirty && <small className="muted">{n('saveFirst')}</small>}</div><div>
      {!dirty && !pending && <ExternalLink href={`${web}/${locale}/gazeta/preview/${initial.id}`}>{t('preview')}</ExternalLink>}
      {canPublish ? <><Button type="button" variant="secondary" disabled={dirty || pending || !selected.length} onClick={() => setAction('publish')}>{t('publish')}</Button>{!!initial.publishedRevision && <Button type="button" variant="danger" disabled={dirty || pending} onClick={() => setAction('unpublish')}>{t('unpublish')}</Button>}</> : <small>{n('publishPermission')}</small>}
    </div></footer>
    {chosen && <Modal title={n('photo')} onClose={() => { if (!uploading) setPhotoId(null) }}><NewspaperPhoto value={chosen.photo || { ...emptyPhoto(), ...(source(chosen)?.media || {}) }} onChange={photo => changeItem(chosen.id, photo)} onBusy={busy => setUploads(v => ({ ...v, article: busy }))} disabled={!canEdit}/><Button type="button" variant="secondary" disabled={uploading} onClick={() => { changeItem(chosen.id); setPhotoId(null) }}>{n('resetPhoto')}</Button></Modal>}
    {action && <Modal title={n(action === 'publish' ? 'publishTitle' : 'unpublishTitle')} onClose={() => { if (!publish.isPending) setAction(null) }}><p>{n(action === 'publish' ? 'publishLead' : 'unpublishLead')}</p>{publish.error && <ErrorPanel error={publish.error}/>}<div className="form-actions"><Button type="button" variant="secondary" disabled={publish.isPending} onClick={() => setAction(null)}>{t('cancel')}</Button><Button type="button" variant={action === 'unpublish' ? 'danger' : 'primary'} disabled={publish.isPending} onClick={() => publish.mutate()}>{t(action)}</Button></div></Modal>}
  </>
}
