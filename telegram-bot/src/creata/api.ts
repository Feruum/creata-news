export class ServiceAPI {
  constructor(readonly origin:string,private token:string){}
  async request<T>(path:string,body?:unknown,method=body===undefined?'GET':'POST'):Promise<T>{
    const response=await fetch(this.origin.replace(/\/$/,'')+'/api/creata/v1'+path,{method,headers:{Authorization:'Bearer '+this.token,'Content-Type':'application/json'},body:body===undefined?undefined:JSON.stringify(body),signal:AbortSignal.timeout(30000)})
    const data=await response.json().catch(()=>({error:'Invalid API response'})) as {error?:string}
    if(!response.ok)throw new Error(data.error||'API request failed')
    return data as T
  }
}
