import { NextResponse,type NextRequest } from 'next/server'
import { isLocale } from './lib/i18n'
export function proxy(request:NextRequest){const headers=new Headers(request.headers),locale=request.nextUrl.pathname.split('/')[1];headers.delete('x-creata-locale');headers.delete('x-creata-path');if(isLocale(locale)){headers.set('x-creata-locale',locale);headers.set('x-creata-path',request.nextUrl.pathname)}const response=NextResponse.next({request:{headers}});if(isLocale(locale))response.cookies.set('creata-locale',locale,{path:'/',sameSite:'lax',maxAge:31536000,secure:request.nextUrl.protocol==='https:'});return response}
export const config={matcher:['/((?!api|_next|favicon|robots|sitemap|fonts|news).*)']}
