import type { NextRequest } from 'next/server'
import { apiOrigin } from '@/lib/api'
import {demoMode} from '@/demo/config'
export const dynamic='force-dynamic'
async function forward(request:NextRequest,{params}:{params:Promise<{path:string[]}>}) {
  const{path}=await params
  if(path.some(segment=>segment==='..'||segment==='.'||segment.includes('/')||segment.includes('\\')))return Response.json({error:'Invalid API path'},{status:400})
  if(demoMode){if(path[0]!=='v1')return Response.json({error:'Not found'},{status:404});const{demoHttp}=await import('@/demo/api');return demoHttp(request,'/'+path.slice(1).map(encodeURIComponent).join('/')+request.nextUrl.search)}
  const target=new URL(`/api/creata/${path.map(encodeURIComponent).join('/')}`,apiOrigin());target.search=request.nextUrl.search
  const headers=new Headers(request.headers);headers.delete('host');headers.delete('content-length');headers.delete('accept-encoding');headers.set('X-Forwarded-Host',request.nextUrl.host);headers.set('X-Forwarded-Proto',request.nextUrl.protocol.replace(':',''))
  try{
    const response=await fetch(target,{method:request.method,headers,body:['GET','HEAD'].includes(request.method)?undefined:await request.arrayBuffer(),redirect:'manual',cache:'no-store',signal:AbortSignal.timeout(20000)})
    const outputHeaders=new Headers(response.headers);outputHeaders.delete('content-length');outputHeaders.delete('content-encoding');outputHeaders.delete('set-cookie');for(const cookie of response.headers.getSetCookie())outputHeaders.append('set-cookie',cookie);outputHeaders.set('Cache-Control','private, no-store')
    return new Response(request.method==='HEAD'?null:response.body,{status:response.status,headers:outputHeaders})
  }catch{return Response.json({error:'Creata API unavailable',code:'api_unavailable'},{status:502,headers:{'Cache-Control':'no-store'}})}
}
export {forward as GET,forward as POST,forward as PATCH,forward as PUT,forward as DELETE,forward as OPTIONS,forward as HEAD}
