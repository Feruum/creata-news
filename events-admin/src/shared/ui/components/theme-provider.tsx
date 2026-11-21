import { useEffect, useState } from 'react'
import { readBrowserSetting, writeBrowserSetting } from '@/shared/lib/browser-storage'
import { type Theme, ThemeProviderContext } from '../theme-provider-context'

type ThemeProviderProps = {
  children: React.ReactNode
  defaultTheme?: Theme
  storageKey?: string
}

export function ThemeProvider({
  children,
  defaultTheme = 'system',
  storageKey = 'vite-ui-theme',
  ...props
}: ThemeProviderProps) {
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = readBrowserSetting(storageKey)
    return saved === 'light' || saved === 'dark' || saved === 'system' ? saved : defaultTheme
  })

  useEffect(() => {
    const root = window.document.documentElement

    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const apply = () => {
      root.classList.remove('light', 'dark')
      root.classList.add(theme === 'system' ? (media.matches ? 'dark' : 'light') : theme)
    }
    apply()
    if (theme !== 'system') return
    media.addEventListener('change', apply)
    return () => media.removeEventListener('change', apply)
  }, [theme])

  const value = {
    theme,
    setTheme: (theme: Theme) => {
      writeBrowserSetting(storageKey, theme)
      setTheme(theme)
    }
  }

  return (
    <ThemeProviderContext.Provider {...props} value={value}>
      {children}
    </ThemeProviderContext.Provider>
  )
}
