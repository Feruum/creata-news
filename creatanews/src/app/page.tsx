import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { isLocale } from '@/lib/i18n'
export default async function Index(){const locale=(await cookies()).get('creata-locale')?.value;redirect(`/${locale&&isLocale(locale)?locale:'ru'}`)}
