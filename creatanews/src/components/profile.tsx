'use client'

import {useEffect,useState,type FormEvent} from 'react'
import Image from 'next/image'
import Link from 'next/link'
import {useRouter} from 'next/navigation'
import {ArrowDown,ArrowUpRight,Bookmark,Check,LogOut,Rss,Send,Sunrise,UserRound} from 'lucide-react'
import type {Category,Locale,User} from '@/lib/models'
import {copy,localePath,localize} from '@/lib/i18n'
import {storyCopy} from '@/lib/story-copy'
import {LaunchSubmissions} from '@/features/launches/launch-submissions'
import {demoCopy} from '@/demo/config'
import {useAuth} from './auth-context'
import {StateMessage} from './state-message'
import styles from './profile.module.css'

const onboardingCopy={
  ru:{interests:'Интересы',delivery:'Рассылка',details:'О вас'},
  kk:{interests:'Қызығушылықтар',delivery:'Таратылым',details:'Сіз туралы'},
  en:{interests:'Interests',delivery:'Delivery',details:'About you'},
}
const stages=['exploring','trying','building'] as const
type Action='save'|'login'|'logout'|'telegram'

export function Profile({locale,categories,returnTo}: {locale:Locale;categories:Category[];returnTo?:string}) {
  const router=useRouter()
  const {auth,loading,error,demo,refresh,request}=useAuth()
  const [draft,setDraft]=useState<User|null>(null)
  const [busy,setBusy]=useState<Action|null>(null)
  const [feedback,setFeedback]=useState<{action:Action;error:boolean}|null>(null)
  const [botLink,setBotLink]=useState('')
  const t=copy[locale],l=onboardingCopy[locale]
  const signInTarget=encodeURIComponent(returnTo ?? localePath(locale,'/profile'))
  const steps=[{id:'profile-interests',label:l.interests},{id:'profile-delivery',label:l.delivery},{id:'profile-details',label:l.details}]

  useEffect(()=>{setDraft(auth.user)},[auth.user])
  useEffect(()=>{
    // Next may append the fragment to a profile URL cached with a fragment already.
    const {pathname,search,hash}=window.location
    if(hash!=='#digest'&&hash.endsWith('#digest')){
      window.history.replaceState(null,'',`${pathname}${search}#digest`)
    }
  },[])

  async function perform(action:Action,work:()=>Promise<void>){
    setBusy(action)
    setFeedback(null)
    try{
      await work()
      if(action==='save')setFeedback({action,error:false})
    }catch{
      setFeedback({action,error:true})
    }finally{
      setBusy(null)
    }
  }

  function save(event:FormEvent<HTMLFormElement>){
    event.preventDefault()
    if(!draft)return
    const {name,locale,stage,topics,frequency,paused}=draft
    void perform('save',async()=>{
      await request<User>('/account',{method:'PATCH',body:JSON.stringify({name,locale,stage,topics,frequency,paused})})
      await refresh()
    })
  }

  function login(){
    void perform('login',async()=>{
      if(demo)await request('/auth/demo-login',{method:'POST'})
      else await request('/auth/dev-login',{method:'POST',body:JSON.stringify({role:'reader'})})
      await refresh()
      if(returnTo)router.replace(returnTo)
    })
  }

  function logout(){
    void perform('logout',async()=>{
      await request('/auth/logout',{method:'POST'})
      setBotLink('')
      await refresh()
    })
  }

  function telegram(){
    void perform('telegram',async()=>{
      const result=await request<{url:string}>('/account/telegram-link',{method:'POST'})
      setBotLink(result.url)
    })
  }

  function status(action:Action){
    if(feedback?.action!==action)return null
    return <p className={`inline-message${feedback.error?' error':''}`} role={feedback.error?'alert':'status'}>
      {feedback.error?t.operationFailed:t.savedChanges}
    </p>
  }

  if(loading)return <div className='state-message' role='status'><span className='state-mark'>✳</span><p>{t.loading}</p></div>
  if(error)return <StateMessage locale={locale} error/>

  return <section id='digest' className={styles.page} aria-labelledby='profile-heading'>
    <header className={styles.hero}>
      <Image className={styles.heroImage} src='/images/home/creata-maker-v1.png' fill sizes='(max-width: 760px) 100vw, 1080px' alt=''/>
      <div className={styles.heroContent}>
        <span className={styles.sunrise}><Sunrise size={23} aria-hidden='true'/></span>
        <h1 id='profile-heading'>{t.digest}</h1>
        <p>{t.signInText}</p>
      </div>
    </header>

    {!auth.user?<section className={styles.login} aria-labelledby='profile-sign-in'>
      <h2 id='profile-sign-in'>{t.signIn}</h2>
      <p>{t.loginNeeded}</p>
      <div className={styles.loginActions}>
        {auth.providers.telegram&&<a className='button button-accent' href={`/api/creata/v1/auth/telegram/start?returnTo=${signInTarget}`}><Send size={16} aria-hidden='true'/>{t.telegram}<ArrowUpRight size={16} aria-hidden='true'/></a>}
        {auth.providers.google&&<a className='button button-outline' href={`/api/creata/v1/auth/google/start?returnTo=${signInTarget}`}>G · {t.google}<ArrowUpRight size={16} aria-hidden='true'/></a>}
        {(demo||auth.devAuth)&&<button type='button' className={demo?'button button-accent':'button button-outline'} disabled={busy!==null} onClick={login}>
          {demo?demoCopy[locale].login:'Local development sign-in'}<ArrowUpRight size={16} aria-hidden='true'/>
        </button>}
      </div>
      {!demo&&!auth.providers.google&&!auth.providers.telegram&&<p>{t.authUnavailable}</p>}
      {status('login')}
    </section>:draft&&<>
      <nav className={styles.steps} aria-label={t.preferences}>
        <ol>{steps.map(({id,label})=><li key={id}><a href={`#${id}`}>
          <span>{label}</span><ArrowDown size={14} aria-hidden='true'/>
        </a></li>)}</ol>
      </nav>
      <div className={styles.layout}>
        <div className={styles.settings}>
          <form className={styles.form} method='post' aria-label={t.preferences} onSubmit={save}>
            <section id='profile-interests' className={styles.group} aria-labelledby='profile-interests-title' tabIndex={-1}>
              <h2 id='profile-interests-title'>{l.interests}</h2>
              <fieldset className={styles.fieldset}>
                <legend>{t.interests}</legend>
                <div className={styles.choices}>
                  {categories.map(category=><label className={styles.choice} key={category.id}>
                    <input type='checkbox' name='topics' value={category.id} checked={draft.topics.includes(category.id)} onChange={event=>setDraft({...draft,topics:event.target.checked?[...draft.topics,category.id]:draft.topics.filter(id=>id!==category.id)})}/>
                    <span>{localize(category.name,locale)}</span>
                  </label>)}
                </div>
              </fieldset>
              <fieldset className={styles.fieldset}>
                <legend>{t.stage}</legend>
                <div className={styles.choices}>
                  {stages.map(stage=><label className={styles.choice} key={stage}>
                    <input type='radio' name='stage' value={stage} checked={draft.stage===stage} onChange={()=>setDraft({...draft,stage})}/>
                    <span>{t[stage]}</span>
                  </label>)}
                </div>
              </fieldset>
            </section>

            <section id='profile-delivery' className={styles.group} aria-labelledby='profile-delivery-title' tabIndex={-1}>
              <h2 id='profile-delivery-title'>{l.delivery}</h2>
              <div className={styles.fieldGrid}>
                <div className={styles.field}>
                  <label htmlFor='profile-frequency'>{t.frequency}</label>
                  <select id='profile-frequency' name='frequency' value={draft.frequency} onChange={event=>setDraft({...draft,frequency:event.target.value as User['frequency']})}>
                    <option value='daily'>{t.daily}</option><option value='weekly'>{t.weekly}</option>
                  </select>
                </div>
                <label className={`${styles.choice} ${styles.pause}`}>
                  <input type='checkbox' name='paused' checked={draft.paused} onChange={event=>setDraft({...draft,paused:event.target.checked})}/>
                  <span>{t.paused}</span>
                </label>
              </div>
            </section>

            <section id='profile-details' className={styles.group} aria-labelledby='profile-details-title' tabIndex={-1}>
              <h2 id='profile-details-title'>{l.details}</h2>
              <div className={styles.fieldGrid}>
                <div className={styles.field}>
                  <label htmlFor='profile-name'>{t.name}</label>
                  <input id='profile-name' name='name' autoComplete='name' value={draft.name} required maxLength={120} onChange={event=>setDraft({...draft,name:event.target.value})}/>
                </div>
                <div className={styles.field}>
                  <label htmlFor='profile-locale'>{t.language}</label>
                  <select id='profile-locale' name='locale' value={draft.locale} onChange={event=>setDraft({...draft,locale:event.target.value as Locale})}>
                    <option value='ru'>Русский</option><option value='kk'>Қазақша</option><option value='en'>English</option>
                  </select>
                </div>
              </div>
            </section>

            <div className={styles.save}>
              <button type='submit' className='button button-accent' disabled={busy!==null}>
                {busy==='save'?t.saving:t.saveChanges}<Check size={17} aria-hidden='true'/>
              </button>
              {status('save')}
            </div>
          </form>
          {!demo&&<LaunchSubmissions key={draft.id} locale={locale} categories={categories}/>}
        </div>

        <aside className={styles.side} aria-label={t.profile}>
          <section className={styles.account} aria-labelledby='profile-account-title'>
            <div className={styles.accountIdentity}>
              <span className={styles.sideIcon}><UserRound size={19} aria-hidden='true'/></span>
              <div><p>{t.profile}</p><h2 id='profile-account-title'>{auth.user.name}</h2></div>
            </div>
            <Link href={localePath(locale,'/following')}><Rss size={16} aria-hidden='true'/>{storyCopy[locale].mine}<ArrowUpRight size={15} aria-hidden='true'/></Link>
            <Link href={localePath(locale,'/saved')}><Bookmark size={16} aria-hidden='true'/>{t.saved}<ArrowUpRight size={15} aria-hidden='true'/></Link>
            <button type='button' disabled={busy!==null} onClick={logout}><LogOut size={16} aria-hidden='true'/>{t.signOut}</button>
            {status('logout')}
          </section>
          <section className={styles.telegram} aria-labelledby='profile-telegram-title'>
            <span className={styles.sideIcon}>{draft.telegramLinked?<Check size={19} aria-hidden='true'/>:<Send size={19} aria-hidden='true'/>}</span>
            <h2 id='profile-telegram-title'>{draft.telegramLinked?t.telegramLinked:t.linkTelegram}</h2>
            <p>{demo?demoCopy[locale].telegram:t.linkHint}</p>
            {!demo&&(botLink?<a href={botLink} target='_blank' rel='noopener noreferrer' className='button button-accent'>{t.openBot}<ArrowUpRight size={16} aria-hidden='true'/></a>:<button type='button' className='button button-outline' onClick={telegram} disabled={busy!==null}>{draft.telegramLinked?t.openBot:t.linkTelegram}</button>)}
            {status('telegram')}
          </section>
        </aside>
      </div>
    </>}
  </section>
}
