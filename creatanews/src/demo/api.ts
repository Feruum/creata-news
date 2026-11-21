import {ApiError,type Article,type Author,type Category,type Launch,type Opportunity,type SitePage,type NewspaperPublication} from '@creata/contracts'
import source from './content.json'
const data=source as unknown as {articles:Article[];authors:Author[];categories:Category[];opportunities:Opportunity[];launches:Launch[];pages:SitePage[];newspapers:NewspaperPublication[]}
function missing():never{throw new ApiError(404,'Demo resource not found','demo_not_found')}
export function readDemoPublic(path:string):unknown{
 if(!path.startsWith('/public/'))return missing()
 const url=new URL(path,'https://creata.demo'),parts=url.pathname.split('/').filter(Boolean).map(decodeURIComponent),kind=parts[1],id=parts[2],q=url.searchParams
 const locale=(['ru','kk','en'].includes(q.get('locale')||'')?q.get('locale'):'ru') as 'ru'|'kk'|'en'
 const list=(items:unknown[])=>{const offset=Math.max(0,Number(q.get('offset'))||0),limit=Math.min(1000,Math.max(1,Number(q.get('limit'))||1000));return{items:items.slice(offset,offset+limit),total:items.length}}
 const find=(items:{id:string;slug?:string}[])=>items.find(item=>item.id===id||item.slug===id)||missing()
 let result:unknown
 if(kind==='articles'||kind==='popular'){
  let items=data.articles.filter(article=>article.status==='published'&&article.title[locale])
  if(id)result=find(items)
  else{
   for(const [parameter,field]of [['category','categoryId'],['authorId','authorId'],['layer','layer'],['region','region'],['format','format']]as const){const value=q.get(parameter);if(value)items=items.filter(item=>item[field]===value)}
   const query=q.get('q')?.trim().toLocaleLowerCase();if(query)items=items.filter(item=>[item.title[locale],item.lead[locale],item.body[locale],item.tags.join(' ')].join(' ').toLocaleLowerCase().includes(query))
   result=list(kind==='popular'?items.slice(0,6):items)
  }
 }else if(kind==='categories')result=id?find(data.categories):list(data.categories)
 else if(kind==='authors')result=id?find(data.authors):list(data.authors)
 else if(kind==='opportunities'){
  let items=data.opportunities.filter(item=>item.status==='approved'&&!!item.title[locale])
  for(const [parameter,field]of [['type','type'],['country','country'],['region','country'],['category','categoryId']]as const){const value=q.get(parameter);if(value)items=items.filter(item=>item[field]===value)}
  result=id?find(items):list(items)
 }else if(kind==='launches'){
  if(parts[3]==='comments')result={items:[],total:0}
  else if(id)result=find(data.launches)
  else{let items=data.launches;const query=q.get('q')?.trim().toLowerCase();if(query)items=items.filter(item=>(item.name+' '+item.tagline[locale]).toLowerCase().includes(query));if(q.get('category'))items=items.filter(item=>item.categoryId===q.get('category'));if(q.get('sort')==='popular')items=[...items].sort((a,b)=>(b.votes||0)-(a.votes||0));result=list(items)}
 }else if(kind==='pages')result=data.pages.find(page=>page.slug===id)||missing()
 else if(kind==='newspapers')result=id?find(data.newspapers):list(data.newspapers)
 else return missing()
 return structuredClone(result)
}

/** No proxying of admin/auth/delivery operations in a public demo deployment. */
export async function demoHttp(request:Request,path:string):Promise<Response>{
 try{
  if(path==='/public/events'&&request.method==='POST')return Response.json({ok:true,demo:true},{status:202})
  if(path==='/auth/me'&&request.method==='GET')return Response.json({user:null,providers:{google:false,telegram:false},demo:true})
  if(request.method!=='GET'&&request.method!=='HEAD')throw new ApiError(403,'This operation is unavailable in the frontend demo','demo_read_only')
  const data=readDemoPublic(path)
  return request.method==='HEAD'?new Response(null,{status:200}):Response.json(data,{headers:{'Cache-Control':'no-store'}})
 }catch(error){const status=error instanceof ApiError?error.status:500;return Response.json({error:error instanceof Error?error.message:'Demo unavailable',code:'demo_unavailable'},{status})}
}
