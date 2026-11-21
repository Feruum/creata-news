import { cache } from 'react'
import { headers } from 'next/headers'
import type { SitePage } from '@creata/contracts'
import { apiOrigin, PublicApiError,serverApi } from '@/lib/api'
import {demoMode} from '@/demo/config'

export const getSitePage = cache(async (slug: string, preview = false): Promise<SitePage> => {
 if(demoMode){if(preview)throw new PublicApiError(404,'Preview unavailable in demo');return serverApi<SitePage>(`/public/pages/${encodeURIComponent(slug)}`)}
 const requestHeaders: Record<string,string> = {}
 if(preview) requestHeaders.Cookie=(await headers()).get('cookie')||''
 const res=await fetch(`${apiOrigin()}/api/creata/v1/${preview?'admin':'public'}/pages/${encodeURIComponent(slug)}`,{cache:'no-store',headers:requestHeaders,signal:AbortSignal.timeout(10000)})
 if(!res.ok)throw new PublicApiError(res.status,`Page API returned ${res.status}`)
 return res.json()
})
