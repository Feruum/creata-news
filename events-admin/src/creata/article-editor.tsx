import { PublicationPostizOptions, PostizDeliveries } from './postiz'
import type { schemas } from '@creata/contracts'
import {applicationUrls} from './demo'
import { useEffect, useState } from 'react'
import { Link, useParams, useSearch } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, articlePhotoLayout } from '@creata/contracts'
import { ArrowLeft, Clock3, ExternalLink as ExternalIcon, Save } from 'lucide-react'
import { useAdmin, useCategories, useStaff } from './hooks'
import { allowedArticleActions, articleFields, canPublishArticle, csv, emptyText, localized } from './domain'
import { Badge, Button, ErrorPanel, Field, Loading, LocaleTabs, Modal, PageHead, SavedIndicator } from './ui'
import { formatDate, locales, type CopyKey } from './i18n'
import type { Article, ArticleAction, Author, List, Locale, Media, Revision } from './types'
import { ArticleResearch } from './pipeline-progress'
import { ArticleBodyField } from './article-body-field'
import { ArticleSources } from './article-sources'
import { ArticleReviewNotes } from './article-review-notes'
import { ArticleLivePreview } from './article-live-preview'
import { ArticleCoverPreview, ArticlePhotosPanel } from './article-photos'
import { ArticleStoryFields } from './article-story-fields'
import { ArticleFollowUp } from './article-follow-up'
import { articleFollowUpKey, storyFollowUpsKey } from './story-follow-up'
import { revisionFields, revisionLines, type RevisionField, type RevisionLine } from './revision-comparison'

const editorTabs = [
 { id: 'sources', label: 'facts', mobileOnly: true },
 { id: 'text', label: 'body' },
 { id: 'story', label: 'storyDevelopment' },
 { id: 'media', label: 'media', mobileOnly: true },
] as const satisfies ReadonlyArray<{ id: string; label: CopyKey; mobileOnly?: boolean }>
type EditorTab = typeof editorTabs[number]['id']
type EditableArticle = Article & { storySlug?: string }

export function ArticleEditor(){
 const {articleId}=useParams({strict:false}) as {articleId:string};const {api,locale}=useAdmin()
 const search = useSearch({ strict: false }) as { tab?: string }
 const [lang,setLang]=useState<Locale>(locale)
 const [tab,setTab]=useState<EditorTab>(search.tab === 'story' ? 'story' : 'text')
 const [livePreview,setLivePreview]=useState(false)
 const query=useQuery({queryKey:['article',articleId],queryFn:()=>api<EditableArticle>(`/admin/articles/${encodeURIComponent(articleId)}`)})
 if(query.isPending)return <Loading/>;if(query.error)return <ErrorPanel error={query.error} retry={()=>void query.refetch()}/>
 return <ArticleEditorForm key={`${query.data.id}:${query.data.revision}`} initial={query.data} reload={()=>void query.refetch()}
   lang={lang} setLang={setLang} tab={tab} setTab={setTab} livePreview={livePreview} togglePreview={()=>setLivePreview(value=>!value)}/>
}

function RevisionCompare({ article, onClose }: { article: Article; onClose: () => void }) {
 const { api, t, locale } = useAdmin()
 const data = useQuery({ queryKey: ['revisions', article.id], queryFn: () => api<List<Revision>>(`/admin/articles/${encodeURIComponent(article.id)}/revisions`) })
 const [version, setVersion] = useState<number>()
 const [lang, setLang] = useState<Locale>(locale)
 const [field, setField] = useState<RevisionField>('body')
 const revisions = data.data?.items ?? []
 const previous = revisions.find(r => r.revision === version) ?? revisions.find(r => r.revision < article.revision) ?? revisions[0]
 const left = previous ? revisionLines(previous, field, lang, locale) : []
 const right = revisionLines(article, field, lang, locale)

 function paragraphs(lines: RevisionLine[], other: RevisionLine[]) {
   const key = (line: RevisionLine) => `${line.updateId ?? ''}\n${line.text}`
   const otherLines = new Set(other.map(key))
   return lines.map((line, index) => <div className={line.text && !otherLines.has(key(line)) ? 'diff-line' : undefined} key={index}>
     {line.href ? <a href={line.href} target='_blank' rel='noopener noreferrer'>{line.text}</a> : line.text || '\u00a0'}
   </div>)
 }

 return <Modal title={t('compare')} onClose={onClose} wide>
   {data.isPending ? <Loading/> : data.error ? <ErrorPanel error={data.error}/> : <>
     <div className='revision-controls'>
       <LocaleTabs value={lang} onChange={setLang}/>
       <select aria-label={t('previous')} value={previous?.revision ?? ''} onChange={event => setVersion(Number(event.target.value))}>
         {revisions.map(r => <option value={r.revision} key={r.revision}>{t('revision')} {r.revision} · {formatDate(r.createdAt, locale, true)}</option>)}
       </select>
       <select aria-label={t('comparisonSection')} value={field} onChange={event => setField(event.target.value as RevisionField)}>
         {Object.entries(revisionFields).map(([value, label]) => <option value={value} key={value}>{t(label)}</option>)}
       </select>
     </div>
     {previous ? <div className='revision-grid'>
       {[{ version: previous, lines: left, other: right, label: t('previous') }, { version: article, lines: right, other: left, label: t('current') }].map(side => <section key={side.label}>
         <div className='revision-label'>{side.label} · {side.version.revision}</div>
         <h3 className='comparison-title'>{localized(side.version.title, lang)}</h3>
         <div className='revision-content'>{side.lines.length ? paragraphs(side.lines, side.other) : '—'}</div>
       </section>)}
     </div> : <p className='callout'>{t('emptyTitle')}</p>}
   </>}
 </Modal>
}

function ArticleEditorForm({initial,reload,lang,setLang,tab,setTab,livePreview,togglePreview}:{
 initial:EditableArticle;reload:()=>void;lang:Locale;setLang:(locale:Locale)=>void;tab:EditorTab;setTab:(tab:EditorTab)=>void;livePreview:boolean;togglePreview:()=>void
}){
 const [postizTargets, setPostizTargets] = useState<schemas['PostizTarget'][] | undefined>([])
 const {api,t,locale,session,toast}=useAdmin();const client=useQueryClient();const categories=useCategories();const authors=useQuery({queryKey:['authors'],queryFn:()=>api<List<Author>>('/admin/authors')})
 const staff=useStaff();const [article,setArticle]=useState<Article>(()=>structuredClone(initial)),[history,setHistory]=useState(false),[action,setAction]=useState<ArticleAction|null>(null),[note,setNote]=useState(''),[schedule,setSchedule]=useState(''),[uploaded,setUploaded]=useState(false)
 const [media,setMedia]=useState<Media>(article.media||{url:'',caption:'',credit:'',rights:''})
 const { cover } = articlePhotoLayout({ media, photos: article.photos })
 const dirty=JSON.stringify(articleFields(article))!==JSON.stringify(articleFields(initial))
 const user=session!.user!;const canEdit=user.roles.includes('chief_editor')||(user.roles.includes('editor')&&user.verticals.includes(article.categoryId))||(user.roles.includes('author')&&article.authorId===user.id)
 const permitted=allowedArticleActions(user,article)
 const missingTranslations=locales.map(language=>({
   language, fields:(['title','body'] as const).filter(field=>!article[field]?.[language]?.trim()),
 })).filter(item=>item.fields.length)
 const onSuccess=(data:unknown)=>{const a=data as Article;if(a?.id&&a.revision){setArticle(structuredClone(a));setMedia(a.media||{url:'',caption:'',credit:'',rights:''});client.setQueryData(['article',a.id],a)}void client.invalidateQueries({queryKey:['article',article.id]});void client.invalidateQueries({queryKey:['postiz-deliveries',article.id]});void client.invalidateQueries({queryKey:['articles']});void client.invalidateQueries({queryKey:['analytics']});void client.invalidateQueries({queryKey:['revisions',article.id]});void client.invalidateQueries({queryKey:articleFollowUpKey(article.id)});void client.invalidateQueries({queryKey:storyFollowUpsKey});toast(t('saved'));setAction(null);setNote('')}
 const save=useMutation({mutationFn:()=>api<Article>(`/admin/articles/${encodeURIComponent(article.id)}`,{method:'PATCH',body:JSON.stringify({...articleFields(article),expectedRevision:initial.revision})}),onSuccess})
 const act=useMutation({mutationFn:()=>api<Article>(`/admin/articles/${encodeURIComponent(article.id)}/actions`,{method:'POST',body:JSON.stringify({action,expectedRevision:initial.revision,postizTargets:(action==='publish'||action==='schedule')?postizTargets:undefined,note:note.trim()||undefined,scheduledAt:action==='schedule'&&schedule?new Date(schedule).toISOString():undefined})}),onSuccess})
 const upload=useMutation({mutationFn:async(file:File)=>{const form=new FormData();form.set('file',file);form.set('credit',media.credit);form.set('rights',media.rights);form.set('caption',media.caption);return api<Media>('/admin/media',{method:'POST',body:form})},onSuccess:data=>{const value={...media,...data};setMedia(value);setArticle(prev=>({...prev,media:value}));setUploaded(true);toast(t('saved'))}})
 const update=(key:string,value:unknown)=>setArticle(prev=>({...prev,[key]:value}));const updateText=(key:'title'|'lead'|'body'|'summary'|'actions',value:string)=>update(key,{...(article[key]||emptyText()),[lang]:value})
 const pending=save.isPending||act.isPending;const failure=save.error||act.error;const conflict=failure instanceof ApiError&&failure.status===409
 const storyTab=tab==='story'
 const updates=article.storyUpdates ?? []
 const updateDateMissing=updates.some(update=>!update.occurredAt)
 useEffect(()=>{if(!dirty)return;const protect=(e:BeforeUnloadEvent)=>{e.preventDefault();e.returnValue=''};window.addEventListener('beforeunload',protect);return()=>window.removeEventListener('beforeunload',protect)},[dirty])
 return <><Link to='/articles' className='back-link'><ArrowLeft size={14}/>{t('back')}</Link><PageHead title={localized(initial.title,locale)||t('untitled')} description={`${t('revision')} ${initial.revision} · ${formatDate(initial.updatedAt,locale,true)}`} action={<><Badge status={article.status}/><Button variant='secondary' onClick={()=>setHistory(true)}><Clock3 size={14}/>{t('revisions')}</Button><a className='button secondary' target='_blank' rel='noopener noreferrer' href={`${applicationUrls.web}/${lang}/preview/${encodeURIComponent(article.id)}`}>{t('preview')}<ExternalIcon size={13}/></a></>}/>
 {failure&&<ErrorPanel error={failure}/>} {conflict&&<div className='callout section-spacing'><p>{t('conflict')}</p><Button className='small' variant='secondary' onClick={reload}>{t('reload')}</Button></div>}
 <div className='editor-tabs segment'>{editorTabs.map(item=><button key={item.id} type='button' data-mobile-only={'mobileOnly' in item}
   className={tab===item.id?'active':''} aria-pressed={tab===item.id} onClick={()=>setTab(item.id)}>{t(item.label)}</button>)}</div>
 <div className='editor-grid' data-live-preview={livePreview}>
 <aside className='editor-side' data-visible={tab==='sources'}>
   <ArticleSources sources={article.sources ?? []} disabled={!canEdit} onChange={sources=>update('sources',sources)}/>
   <ArticleReviewNotes notes={article.reviewNotes} locale={locale} ecosystem={article.layer==='eco'}/>
   <ArticleResearch articleId={article.id} opportunityIds={article.opportunityIds} researchBrief={article.researchBrief}/>
 </aside>
 <section className='panel editor-main' data-visible={tab==='text'||storyTab}>
   <header className='article-editor-toolbar'>
     <h2>{t(storyTab?'storyDevelopment':'body')}</h2>
     <LocaleTabs value={lang} onChange={setLang}/>
     <Button type='button' variant='secondary' className='small' aria-expanded={livePreview}
       aria-controls={livePreview?'article-live-preview':undefined} onClick={togglePreview}>{t('livePreview')}</Button>
   </header>
   {!storyTab&&!!missingTranslations.length&&<section className='article-readiness' aria-label={t('publicationReadiness')}>
     <p>{t('completeTranslations')}</p>
     <ul>{missingTranslations.map(({language,fields})=>{
       const languageLabel=language==='kk'?'ҚАЗ':language.toUpperCase()
       const missingFields=fields.map(field=>t(field)).join(', ')
       return <li key={language}>
         <Button type='button' variant='ghost' className='small' aria-label={`${languageLabel} · ${missingFields}`}
           onClick={()=>setLang(language)}>{languageLabel}</Button>
         <span>{missingFields}</span>
       </li>
     })}</ul>
   </section>}
   <div className={livePreview?'article-editor-split':undefined}>
     {storyTab ? <div className='form-stack'>
       {canPublishArticle(user, initial) && <ArticleFollowUp article={initial} disabled={pending}/>}
       <ArticleStoryFields guide={article.actionGuide} updates={updates} language={lang} disabled={!canEdit||pending}
         onGuideChange={guide=>update('actionGuide',guide)} onUpdatesChange={items=>update('storyUpdates',items)}/>
     </div>
     : <div className='form-stack'>
       <Field label={t('title')}><textarea disabled={!canEdit} className='editor-title' value={article.title?.[lang]||''} onChange={e=>updateText('title',e.target.value)}/></Field>
       <Field label={t('lead')}><textarea disabled={!canEdit} value={article.lead?.[lang]||''} onChange={e=>updateText('lead',e.target.value)}/></Field>
       <ArticleBodyField disabled={!canEdit} language={lang} value={article.body?.[lang]||''} onChange={value=>updateText('body',value)}/>
       <Field label={t('summary')}><textarea disabled={!canEdit} value={article.summary?.[lang]||''} onChange={e=>updateText('summary',e.target.value)}/></Field>
       <Field label={t('actions')}><textarea disabled={!canEdit} value={article.actions?.[lang]||''} onChange={e=>updateText('actions',e.target.value)}/></Field>
     </div>}
     {livePreview&&<ArticleLivePreview article={article} language={lang} label={t('livePreview')} publicWebsite={applicationUrls.web}/>}
   </div>
 </section>
 <aside className='editor-side editor-media' data-visible={tab==='media'}><section className='panel'><h2>{t('media')}</h2><ArticleCoverPreview media={media} photo={cover} language={locale}/><div className='form-stack'><Field label={t('credit')}><input disabled={!canEdit} value={media.credit} onChange={e=>{const next={...media,credit:e.target.value};setMedia(next);if(media.url)update('media',next)}}/></Field><Field label={t('rights')} hint={t('rightsHint')}><input disabled={!canEdit} value={media.rights} onChange={e=>{const next={...media,rights:e.target.value};setMedia(next);if(media.url)update('media',next)}}/></Field><Field label={t('caption')}><input disabled={!canEdit} value={media.caption} onChange={e=>{const next={...media,caption:e.target.value};setMedia(next);if(media.url)update('media',next)}}/></Field><Field label={t('upload')} hint={t('uploadHint')}><input className='upload-input' type='file' accept='image/jpeg,image/png,image/webp,image/gif' disabled={!canEdit||!media.credit.trim()||!media.rights.trim()||upload.isPending} onChange={e=>{const file=e.target.files?.[0];if(file)upload.mutate(file)}}/></Field>{upload.error&&<ErrorPanel error={upload.error}/>}</div></section>
 <ArticlePhotosPanel photos={article.photos ?? []} cover={cover} language={locale}
   disabled={!canEdit || pending} onChange={photos=>update('photos',photos)}/>
 <section className='panel'><h2>{t('settings')}</h2><div className='form-stack'><Field label={t('layer')}><select disabled={!canEdit} value={article.layer} onChange={e=>update('layer',e.target.value)}><option value='world'>{t('world')}</option><option value='eco'>{t('eco')}</option></select></Field><Field label={t('topic')}><select disabled={!canEdit} value={article.categoryId} onChange={e=>update('categoryId',e.target.value)}><option value=''>{t('select')}</option>{categories.data?.items.map(c=><option key={c.id} value={c.id}>{localized(c.name,locale)}</option>)}</select></Field><Field label={t('author')}><select disabled={!canEdit} value={article.authorId} onChange={e=>update('authorId',e.target.value)}><option value=''>{t('select')}</option>{authors.data?.items.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</select></Field><Field label={t('editor')}><select disabled={!canEdit} value={(article as Article&{editorId?:string}).editorId||''} onChange={e=>update('editorId',e.target.value)}><option value=''>{t('select')}</option>{staff.data?.items.filter(u=>u.roles.includes('chief_editor')||(u.roles.includes('editor')&&u.verticals.includes(article.categoryId))).map(u=><option key={u.id} value={u.id}>{u.name}</option>)}</select></Field><Field label={t('region')}><input disabled={!canEdit} value={article.region} onChange={e=>update('region',e.target.value)}/></Field><Field label={t('tags')}><input disabled={!canEdit} value={article.tags?.join(', ')||''} onChange={e=>update('tags',csv(e.target.value))}/></Field><Field label={t('slug')} hint={initial.storySlug?t('storySlugLocked'):undefined}><input disabled={!canEdit||!!initial.storySlug} value={article.slug} onChange={e=>update('slug',e.target.value)}/></Field></div></section></aside></div>
 {(canEdit||user.roles.includes('admin'))&&<PostizDeliveries articleId={initial.id}/>}
 <footer className='editor-controls'><div><Button disabled={!canEdit||!dirty||pending||updateDateMissing} aria-describedby={updateDateMissing?'story-update-date-required':undefined} onClick={()=>save.mutate()}><Save size={14}/>{save.isPending?t('saving'):t('save')}</Button>{save.isSuccess&&!dirty&&<SavedIndicator/>}{uploaded&&dirty&&<small className='muted'>{t('save')}</small>}{updateDateMissing&&<small id='story-update-date-required' className='muted'>{t('storyUpdateDateRequired')}</small>}</div><div className='editor-action-list'>{permitted.map(a=><Button key={a} variant={a==='reject'||a==='unpublish'?'danger':'secondary'} disabled={dirty||pending} onClick={()=>{setAction(a);setNote('');setSchedule('');setPostizTargets(initial.status==='scheduled'?undefined:[])}}>{t(a==='rework'?'requestRework':a)}</Button>)}</div></footer>
 {history&&<RevisionCompare article={article} onClose={()=>setHistory(false)}/>} {action&&<Modal title={t(action==='rework'?'requestRework':action)} onClose={()=>setAction(null)}><form className='form-stack' onSubmit={e=>{e.preventDefault();act.mutate()}}>{(action==='publish'||action==='schedule')&&<PublicationPostizOptions article={initial} value={postizTargets} onChange={setPostizTargets}/>}
 {action==='schedule'&&<Field label={t('scheduledAt')} hint={t('scheduleHint')}><input required type='datetime-local' min={new Date(Date.now()-new Date().getTimezoneOffset()*60000).toISOString().slice(0,16)} value={schedule} onChange={e=>setSchedule(e.target.value)}/></Field>}<Field label={t((action==='publish'||action==='schedule')&&initial.publishedRevision?'correctionNote':'note')} hint={(action==='publish'||action==='schedule')&&initial.publishedRevision?t('correctionHelp'):undefined}><textarea required={action==='rework'||action==='reject'} value={note} placeholder={t('noteHint')} onChange={e=>setNote(e.target.value)}/></Field>{act.error&&<ErrorPanel error={act.error}/>}<div className='form-actions'><Button variant='secondary' type='button' onClick={()=>setAction(null)}>{t('cancel')}</Button><Button disabled={act.isPending}>{t(action==='rework'?'requestRework':action)}</Button></div></form></Modal>}</>
}



