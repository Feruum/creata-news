'use client'

import { useEffect, useRef, type ReactNode } from 'react'

type Connection = { saveData?: boolean; effectiveType?: string; addEventListener?: (event: string, callback: () => void) => void; removeEventListener?: (event: string, callback: () => void) => void }

/** Scoped interaction adapter for the fixed, server-rendered newspaper HTML. */
export function IssuePhotography({ children }: { children: ReactNode }) {
  const root = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const container = root.current
    if (!container) return
    const desk = container.querySelector<HTMLElement>('.desk')!
    const toggle = container.querySelector<HTMLButtonElement>('#wake-all')!
    const hint = container.querySelector<HTMLElement>('.hint')!
    if(!desk||!toggle||!hint)return
    const photos = [...container.querySelectorAll<HTMLElement>('.pic')]
    const live = photos.filter(photo => photo.dataset.loop && photo.querySelector('video'))
    const reduced = matchMedia('(prefers-reduced-motion: reduce)')
    const hover = matchMedia('(hover: hover) and (pointer: fine)')
    const connection = (navigator as Navigator & { connection?: Connection }).connection
    const cleanup: (() => void)[] = []
    const observer = new IntersectionObserver(entries => {
      for (const entry of entries) {
        const photo = entry.target as HTMLElement
        if (entry.intersectionRatio >= .6) { visible.add(photo); if (!hover.matches || all) wake(photo) }
        else { visible.delete(photo); if (!hover.matches || all) sleep(photo, all) }
      }
    }, { threshold: [0, .6] })
    const visible = new Set<HTMLElement>()
    let all = false
    let disposed = false
    const blocked = () => reduced.matches || Boolean(connection?.saveData) || /(^|slow-)2g|3g/.test(connection?.effectiveType || '')
    function wake(photo: HTMLElement) {
      photo.classList.add('is-color')
      const video = photo.querySelector('video')
      if (!video || blocked()) return
      if (!video.getAttribute('src')) video.src = photo.dataset.loop!
      video.muted = true
      photo.dataset.playback = 'requested'
      void video.play().then(() => {
        if (!disposed && photo.dataset.playback === 'requested' && !blocked()) {
          photo.classList.add('is-awake'); photo.setAttribute('aria-pressed', 'true')
        } else video.pause()
      }).catch(() => { /* The still remains visible when playback is unavailable. */ })
    }
    function sleep(photo: HTMLElement, keepColor = false) {
      delete photo.dataset.playback
      photo.classList.remove('is-awake')
      if (!keepColor) photo.classList.remove('is-color')
      if (photo.hasAttribute('aria-pressed')) photo.setAttribute('aria-pressed', 'false')
      photo.querySelector('video')?.pause()
    }
    function updateSettings() {
      all = false; desk.classList.remove('all-awake')
      const noMotion = blocked()
      desk.classList.toggle('motion-off', noMotion)
      toggle.hidden = noMotion; toggle.setAttribute('aria-pressed', 'false'); toggle.textContent = 'Оживить все фото'
      hint.textContent = reduced.matches ? 'движение выключено настройками системы' : noMotion ? 'экономия трафика: фото без движения' : hover.matches ? 'наведите на фото — оно оживёт' : 'фото оживают, когда попадают на экран'
      live.forEach(photo => { sleep(photo); photo.setAttribute('aria-disabled', String(noMotion)) })
      if (!hover.matches) visible.forEach(wake)
    }
    function listen(target: EventTarget, event: string, handler: EventListener) {
      target.addEventListener(event, handler); cleanup.push(() => target.removeEventListener(event, handler))
    }
    for (const photo of photos) {
      observer.observe(photo)
      listen(photo, 'mouseenter', () => { if (hover.matches) wake(photo) })
      listen(photo, 'mouseleave', () => { if (hover.matches && !all && document.activeElement !== photo) sleep(photo) })
    }
    for (const photo of live) {
      photo.setAttribute('role', 'button'); photo.setAttribute('aria-pressed', 'false')
      photo.setAttribute('aria-label', `${photo.querySelector('img')?.alt || 'Фото'}. Включить или остановить движение`)
      listen(photo, 'focus', () => wake(photo))
      listen(photo, 'blur', () => { if (!all) sleep(photo) })
      listen(photo, 'click', () => { if (!blocked()) photo.classList.contains('is-awake') ? sleep(photo) : wake(photo) })
      listen(photo, 'keydown', event => {
        const key = event as KeyboardEvent
        if (key.key === 'Enter' || key.key === ' ') { key.preventDefault(); if (!blocked()) photo.classList.contains('is-awake') ? sleep(photo) : wake(photo) }
      })
    }
    listen(toggle, 'click', () => {
      if (blocked()) return
      all = !all; desk.classList.toggle('all-awake', all)
      toggle.setAttribute('aria-pressed', String(all)); toggle.textContent = all ? 'Усыпить фото' : 'Оживить все фото'
      // Colour all photographs, but stream only visible loops; a 36-page issue
      // must not download every video at once when this control is activated.
      photos.forEach(photo => { if (all) { photo.classList.add('is-color'); if (visible.has(photo)) wake(photo) } else sleep(photo) })
    })
    listen(document, 'visibilitychange', () => {
      if (document.hidden) live.forEach(photo => sleep(photo, all))
      else if (all || !hover.matches) visible.forEach(wake)
    })
    reduced.addEventListener('change', updateSettings); hover.addEventListener('change', updateSettings)
    connection?.addEventListener?.('change', updateSettings)
    updateSettings()
    return () => {
      disposed = true; observer.disconnect(); cleanup.forEach(fn => fn())
      reduced.removeEventListener('change', updateSettings); hover.removeEventListener('change', updateSettings)
      connection?.removeEventListener?.('change', updateSettings)
      live.forEach(photo => sleep(photo))
    }
  }, [])
  return <div ref={root} className="newspaper-issue" lang="ru">{children}</div>
}
