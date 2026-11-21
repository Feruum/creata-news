'use client'

import { useRef, useSyncExternalStore } from 'react'
import { flushSync } from 'react-dom'
import { type Theme, themeStorageKey } from '../lib/theme'

function getTheme(): Theme {
  return document.documentElement.dataset.theme === 'light' ? 'light' : 'dark'
}

function subscribe(onChange: () => void) {
  const synchronize = (event: StorageEvent) => {
    if (event.key !== themeStorageKey && event.key !== null) return
    document.documentElement.dataset.theme = event.newValue === 'light' ? 'light' : 'dark'
    onChange()
  }
  window.addEventListener('creata:theme-change', onChange)
  window.addEventListener('storage', synchronize)
  return () => {
    window.removeEventListener('creata:theme-change', onChange)
    window.removeEventListener('storage', synchronize)
  }
}

export function useTheme() {
  const theme = useSyncExternalStore(subscribe, getTheme, () => 'dark' as const)
  const active = useRef<{ theme: Theme; transition: ViewTransition } | null>(null)

  function toggleTheme(origin: HTMLElement) {
    const next = (active.current?.theme ?? getTheme()) === 'light' ? 'dark' : 'light'
    const root = document.documentElement
    const apply = () => {
      root.dataset.theme = next
      try {
        window.localStorage.setItem(themeStorageKey, next)
      } catch {
        // Theme switching remains available when browser storage is blocked.
      }
      flushSync(() => window.dispatchEvent(new Event('creata:theme-change')))
    }
    active.current?.transition.skipTransition()
    if (!document.startViewTransition || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      apply()
      return
    }

    const bounds = origin.getBoundingClientRect()
    const x = bounds.left + bounds.width / 2
    const y = bounds.top + bounds.height / 2
    const radius = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y))
    root.style.setProperty('--theme-reveal-x', `${x}px`)
    root.style.setProperty('--theme-reveal-y', `${y}px`)
    root.style.setProperty('--theme-reveal-radius', `${Math.ceil(radius)}px`)
    root.dataset.themeTransition = 'true'
    const transition = document.startViewTransition(apply)
    active.current = { theme: next, transition }
    const finish = () => {
      if (active.current?.transition !== transition) return
      active.current = null
      delete root.dataset.themeTransition
      for (const name of ['x', 'y', 'radius']) root.style.removeProperty(`--theme-reveal-${name}`)
    }
    // A hidden tab or a second click can skip a snapshot; the theme still updates.
    void transition.ready.catch(() => {})
    void transition.finished.then(finish, finish)
  }
  return { theme, toggleTheme }
}
