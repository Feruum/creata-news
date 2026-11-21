import type { MetadataRoute } from 'next'
import { siteOrigin } from '@/lib/api'
import {demoMode} from '@/demo/config'
export const dynamic='force-dynamic'
export default function robots():MetadataRoute.Robots{if(demoMode)return{rules:{userAgent:"*",disallow:"/"}};return {rules:{userAgent:'*',allow:'/',disallow:['/api/','/*/preview/','/*/profile','/*/saved','/*/search']},sitemap:`${siteOrigin()}/sitemap.xml`}}
