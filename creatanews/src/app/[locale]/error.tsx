'use client'
import { usePathname } from 'next/navigation'
import { isLocale } from '@/lib/i18n'
import { StateMessage } from '@/components/state-message'
export default function ErrorPage(){const raw=usePathname().split('/')[1];return <StateMessage locale={isLocale(raw)?raw:'ru'} error/>}
