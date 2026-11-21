import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { isLocale } from '@/lib/i18n'

export default async function Page({params}:{params:Promise<{slug:string}>}) {
  const {slug}=await params,saved=(await cookies()).get('creata-locale')?.value
  redirect(`/${saved&&isLocale(saved)?saved:'ru'}/launches/${encodeURIComponent(slug)}`)
}
