'use client'
import { usePathname } from 'next/navigation'
import { copy,isLocale } from '@/lib/i18n'
import { StateMessage } from '@/components/state-message'
export default function NotFound(){const raw=usePathname().split('/')[1],locale=isLocale(raw)?raw:'ru';return <StateMessage locale={locale} title={copy[locale].notFound} text={copy[locale].articleUnavailable} home/>}
