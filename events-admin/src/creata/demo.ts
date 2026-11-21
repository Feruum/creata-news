import type {createApi} from '@creata/contracts'
import type {createPipelineDemo} from '@creata/demo-pipeline'
export const adminDemo=import.meta.env.VITE_CREATA_DEMO_MODE==='1'
export const adminBase=(import.meta.env.VITE_ADMIN_BASE_PATH||'').replace(/\/$/,'')
export const adminPath=(path:string)=>adminBase+(path.startsWith('/')?path:'/'+path)
export const adminRoutePath=(path:string)=>adminBase&&path.startsWith(adminBase+'/')?path.slice(adminBase.length):path
export const applicationUrls={
 web:import.meta.env.VITE_WEB_URL||(adminDemo?'':'http://localhost:3002'),
 editorial:import.meta.env.VITE_EDITORIAL_URL||(adminDemo?'/editorial':'http://localhost:6868'),
 hermes:import.meta.env.VITE_HERMES_ADMIN_URL||(adminDemo?'/hermes':'http://localhost:5174'),
}
let engine:Promise<ReturnType<typeof createPipelineDemo>>|undefined
export function getDemoPipeline(){
 if(!engine)engine=Promise.all([import('@creata/demo-pipeline'),import('../../../creatanews/src/demo/content.json')]).then(([module,seed])=>module.createPipelineDemo(seed.default as unknown as Parameters<typeof createPipelineDemo>[0]))
 return engine
}
export const demoRequest:ReturnType<typeof createApi>=async<T,>(path:string,init:RequestInit={})=>(await getDemoPipeline()).request<T>(path,init)
