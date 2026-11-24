import type { NextRequest } from 'next/server'
import { apiOrigin } from '@/lib/api'
import {demoMode} from '@/demo/config'
export const dynamic='force-dynamic'

const maxBodyBytes = 12 * 1024 * 1024
async function readBody(request: Request) {
  if (!request.body) return undefined
  if (Number(request.headers.get('content-length')) > maxBodyBytes) return null
  const reader = request.body.getReader()
  const chunks: Uint8Array[] = []
  let size = 0
  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      size += value.byteLength
      if (size > maxBodyBytes) {
        void reader.cancel().catch(() => {})
        return null
      }
      chunks.push(value)
    }
  } finally {
    reader.releaseLock()
  }
  const body = new Uint8Array(size)
  let offset = 0
  for (const chunk of chunks) {
    body.set(chunk, offset)
    offset += chunk.byteLength
  }
  return body
}

async function forward(request:NextRequest,{params}:{params:Promise<{path:string[]}>}) {
  const{path}=await params
  if(path.some(segment=>segment==='..'||segment==='.'||segment.includes('/')||segment.includes('\\')))return Response.json({error:'Invalid API path'},{status:400})
  if(path[0]!=='v1'||path[1]==='internal')return Response.json({error:'Not found'},{status:404})
  if(demoMode){const{demoHttp}=await import('@/demo/api');return demoHttp(request,'/'+path.slice(1).map(encodeURIComponent).join('/')+request.nextUrl.search)}
  const target=new URL(`/api/creata/${path.map(encodeURIComponent).join('/')}`,apiOrigin());target.search=request.nextUrl.search
  const headers=new Headers(request.headers);headers.delete('host');headers.delete('content-length');headers.delete('accept-encoding');headers.set('X-Forwarded-Host',request.nextUrl.host);headers.set('X-Forwarded-Proto',request.nextUrl.protocol.replace(':',''))
  headers.delete('X-Creata-SSR-Token')
  try{
    const body = await readBody(request)
    if (body === null) return Response.json({error:'Request body too large'},{status:413})
    const response=await fetch(target,{method:request.method,headers,body,redirect:'manual',cache:'no-store',signal:AbortSignal.timeout(20000)})
    const outputHeaders=new Headers(response.headers);outputHeaders.delete('content-length');outputHeaders.delete('content-encoding');outputHeaders.delete('set-cookie');for(const cookie of response.headers.getSetCookie())outputHeaders.append('set-cookie',cookie);outputHeaders.set('Cache-Control','private, no-store')
    return new Response(request.method==='HEAD'?null:response.body,{status:response.status,headers:outputHeaders})
  }catch{return Response.json({error:'Creata API unavailable',code:'api_unavailable'},{status:502,headers:{'Cache-Control':'no-store'}})}
}
export {forward as GET,forward as POST,forward as PATCH,forward as PUT,forward as DELETE,forward as OPTIONS,forward as HEAD}
