import {apiOrigin} from '@/lib/api'
import {demoMode} from '@/demo/config'
export async function GET(_request:Request,{params}:{params:Promise<{path:string[]}>}){
 const{path}=await params
 if(demoMode)return new Response(null,{status:307,headers:{Location:'/demo-media/'+path.map(encodeURIComponent).join('/')}})
 const res=await fetch(`${apiOrigin()}/legacy-media/${path.map(encodeURIComponent).join('/')}`,{cache:'no-store',redirect:'manual'})
 if(res.status===307&&res.headers.get('location'))return new Response(null,{status:307,headers:{Location:res.headers.get('location')!,'Cache-Control':'no-store'}})
 return Response.json({error:'Media unavailable'},{status:res.status===404?404:503})
}
