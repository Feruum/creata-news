import { Moon, Sun } from 'lucide-react'
import { useAdminTheme, type AdminTheme } from './admin-theme'
import { useAdmin } from './hooks'
import { ErrorPanel, PageHead } from './ui'
import type { CopyKey } from './i18n'

const options: { value: AdminTheme; label: CopyKey }[] = [
  { value: 'light', label: 'themeLight' },
  { value: 'dark', label: 'themeDark' },
  { value: 'cats', label: 'themeCats' },
]

export function AppearanceSettings() {
  const { t } = useAdmin()
  const { theme, setTheme, error, pending } = useAdminTheme()
  return <section className='panel appearance-panel'>
    <h2 id='appearance-heading'>{t('appearance')}</h2>
    <p className='muted'>{t('appearanceHelp')}</p>
    <fieldset className='appearance-options' aria-labelledby='appearance-heading' disabled={pending}>
      {options.map(({ value, label }) => <label className='appearance-option' key={value}>
        <span className={`appearance-swatch appearance-swatch--${value}`} aria-hidden='true'>
          {value === 'light' ? <Sun size={22}/> : value === 'dark' ? <Moon size={22}/> : <span className='appearance-cat-preview'/>}
          <i/><i/><i/>
        </span>
        <span className='appearance-option-label'>{t(label)}<input type='radio' name='admin-appearance' value={value} checked={theme === value}
          onChange={() => setTheme(value)}/></span>
      </label>)}
    </fieldset>
    {pending && <p role='status' className='muted'>{t('saving')}</p>}
    {error && <ErrorPanel error={error}/>}
  </section>
}

export function AppearancePage() {
  const { t } = useAdmin()
  return <><PageHead title={t('settings')}/><AppearanceSettings/></>
}
