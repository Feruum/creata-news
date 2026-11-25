import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Save, Sparkles } from 'lucide-react'
import { adminDemo } from './demo'
import { aiSettingsPatch } from './domain'
import { useAdmin } from './hooks'
import { Button, ErrorPanel, Field, Loading, PageHead } from './ui'
import { TeamAccess } from './team-access'
import { AppearanceSettings } from './appearance-settings'
import type { Settings } from './types'

export function EditorialSettings() {
 const { t, session } = useAdmin()
 return <>
  <PageHead title={t('settings')}/>
  <AppearanceSettings/>
  {session?.user?.roles.includes('admin') && <><SettingsForm kind='editorial'/><TeamAccess/></>}
 </>
}
export function AISettingsPage(){const {t}=useAdmin();return <><PageHead title={t('ai')}/><SettingsForm kind='ai'/></>}
export function RulesPage(){const {t}=useAdmin();return <><PageHead title={t('rules')}/><SettingsForm kind='rules'/></>}
function SettingsForm({kind}:{kind:'editorial'|'ai'|'rules'}){const {api}=useAdmin();const query=useQuery({queryKey:['settings'],queryFn:()=>api<Settings>('/admin/settings')});if(query.isPending)return <Loading/>;if(query.error)return <ErrorPanel error={query.error} retry={()=>void query.refetch()}/>;return <SettingsFields key={`${kind}:${query.data.editorialInstructions}:${query.data.ai.model}:${query.data.ai.dailyBudgetUsd}`} initial={query.data} kind={kind}/>}
function SettingsFields({initial,kind}:{initial:Settings;kind:'editorial'|'ai'|'rules'}) {
 const { api, t, toast } = useAdmin()
 const client = useQueryClient()
 const [settings, setSettings] = useState(() => structuredClone(initial))
 const [secret, setSecret] = useState('')
 const [exaKey, setExaKey] = useState('')
 const [clearExaKey, setClearExaKey] = useState(false)
 const exaConfigured = initial.ai.exaKeyConfigured === true
 const shouldClearExaKey = exaConfigured && clearExaKey
 const mutation = useMutation({
  mutationFn: () => {
   const exa = adminDemo ? undefined : { key: exaKey, clear: shouldClearExaKey }
   const patch = kind === 'ai' ? { ai: aiSettingsPatch(settings.ai, secret, exa) }
    : kind === 'rules' ? { editorialInstructions: settings.editorialInstructions, scoreThreshold: settings.scoreThreshold }
    : { digestHour: settings.digestHour, morningDigestHour: settings.morningDigestHour, timezone: settings.timezone }
   return api<Settings>('/admin/settings', { method: 'PATCH', body: JSON.stringify(patch) })
  },
  onSuccess: saved => {
   setSecret('')
   setExaKey('')
   setClearExaKey(false)
   if (kind === 'ai') client.setQueryData(['settings'], saved)
   void client.invalidateQueries({queryKey:['settings']})
   toast(t('saved'))
  },
 })
 const ai = (key: keyof Settings['ai'], value: string | number) => setSettings(s => ({ ...s, ai: { ...s.ai, [key]: value } }))
 return <form className='settings-grid' style={kind === 'ai' ? undefined : { gridTemplateColumns: 'minmax(0,1fr)' }} onSubmit={e => { e.preventDefault(); if (!mutation.isPending) mutation.mutate() }}>
  <section className='panel'>
   <h2>{t(kind === 'editorial' ? 'settings' : kind)}</h2>
   <div className='form-stack'>
    {kind === 'rules' ? <>
     <Field label={t('instructions')}><textarea required value={settings.editorialInstructions} onChange={e => setSettings({...settings, editorialInstructions: e.target.value})}/></Field>
     <Field label={t('threshold')} hint={t('thresholdHelp')}><input type='number' min='0' max='100' step='1' required value={settings.scoreThreshold} onChange={e => setSettings({...settings, scoreThreshold: Number(e.target.value)})}/></Field>
    </> : kind === 'ai' ? <>
     <Field label={t('endpoint')}><input type='url' value={settings.ai.endpoint} placeholder='https://api.example.com/v1' onChange={e => ai('endpoint', e.target.value)}/></Field>
     <Field label={t('model')}><input value={settings.ai.model} onChange={e => ai('model', e.target.value)} autoComplete='off'/></Field>
     <Field label={t('apiKey')} hint={t('secretHelp')}><input type='password' value={secret} onChange={e => setSecret(e.target.value)} autoComplete='new-password' placeholder={settings.ai.keyConfigured ? '••••••••' : ''}/></Field>
     {!adminDemo && <div className='flex flex-col gap-2'>
      <Field label={t('exaApiKey')} hint={t('exaApiKeyHelp')}>
       <input type='password' value={exaKey} onChange={e => setExaKey(e.target.value)} autoComplete='new-password'
        placeholder={exaConfigured ? '••••••••' : ''} disabled={shouldClearExaKey || mutation.isPending}/>
      </Field>
      {exaConfigured && <label className='checkbox-label'>
       <input type='checkbox' checked={clearExaKey} onChange={e => setClearExaKey(e.target.checked)} disabled={mutation.isPending}/>
       {t('clearExaApiKey')}
      </label>}
     </div>}
     <div className='form-grid'>
      <Field label={t('dailyBudget')}><input type='number' required min='0' step='.01' value={settings.ai.dailyBudgetUsd} onChange={e => ai('dailyBudgetUsd', Number(e.target.value))}/></Field>
      <div/>
      <Field label={t('inputPrice')}><input type='number' required min='0' step='.001' value={settings.ai.inputPricePerMillion} onChange={e => ai('inputPricePerMillion', Number(e.target.value))}/></Field>
      <Field label={t('outputPrice')}><input type='number' required min='0' step='.001' value={settings.ai.outputPricePerMillion} onChange={e => ai('outputPricePerMillion', Number(e.target.value))}/></Field>
     </div>
    </> : <>
     <Field label={t('digestHour')}><input required type='number' min='0' max='23' step='1' value={settings.digestHour} onChange={e => setSettings({...settings, digestHour: Number(e.target.value)})}/></Field>
     {!adminDemo && <Field label={t('morningDigestHour')} hint={t('morningDigestHourHelp')}>
      <input required type='number' min='0' max='23' step='1' value={settings.morningDigestHour ?? 9} onChange={e => setSettings({...settings, morningDigestHour: Number(e.target.value)})}/>
     </Field>}
     <Field label={t('timezone')}><input required value={settings.timezone} onChange={e => setSettings({...settings, timezone: e.target.value})}/></Field>
    </>}
    {mutation.error && <ErrorPanel error={mutation.error}/>}
    <div className='form-actions'><Button disabled={mutation.isPending}><Save size={14}/>{mutation.isPending ? t('saving') : t('save')}</Button></div>
   </div>
  </section>
  {kind === 'ai' && <aside className='panel'>
   <span className='settings-symbol'><Sparkles size={25}/></span>
   <h2>{t('ai')}</h2>
   <span className={`badge ${settings.ai.keyConfigured ? 'approved' : 'not_configured'}`}>{t(settings.ai.keyConfigured ? 'configured' : 'not_configured')}</span>
   <div className='mini-stats section-spacing'>
    <div className='mini-stat'><span>{t('spentToday')}</span><strong>${(initial.ai.spentTodayUsd || 0).toFixed(2)}</strong></div>
    <div className='mini-stat'><span>{t('dailyBudget')}</span><strong>${initial.ai.dailyBudgetUsd.toFixed(2)}</strong></div>
   </div>
  </aside>}
 </form>
}
