import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { isLocale } from '@/lib/i18n'

export default async function Page() {
  const saved=(await cookies()).get('creata-locale')?.value
  redirect(`/${saved&&isLocale(saved)?saved:'ru'}/launches`)
}
