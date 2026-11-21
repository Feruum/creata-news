'use client'

import { Moon, Sun } from 'lucide-react'
import type { Locale } from '@/lib/models'
import { useTheme } from '../hooks/use-theme'

const labels = {
  ru: { light: 'Включить светлую тему', dark: 'Включить тёмную тему' },
  en: { light: 'Switch to light theme', dark: 'Switch to dark theme' },
  kk: { light: 'Ашық тақырыпқа ауысу', dark: 'Қараңғы тақырыпқа ауысу' }
}

export function ThemeToggle({ locale }: { locale: Locale }) {
  const { theme, toggleTheme } = useTheme()
  const label = labels[locale][theme === 'dark' ? 'light' : 'dark']
  return (
    <button
      type='button'
      className='theme-toggle'
      onClick={(event) => toggleTheme(event.currentTarget)}
      aria-label={label}
      title={label}
    >
      <Sun className='theme-sun' size={18} aria-hidden='true' />
      <Moon className='theme-moon' size={18} aria-hidden='true' />
    </button>
  )
}
