/** Compile the existing Vite apps as static same-origin demo entry points. */
import {spawn} from 'node:child_process'
import {cpSync,mkdirSync,readdirSync,statSync,writeFileSync} from 'node:fs'
import {resolve} from 'node:path'
const root=resolve(import.meta.dirname,'..'),generation=Date.now()
const files:string[]=[]
async function build(folder:string,base:string){
 const output=resolve(root,'.runtime/demo-admin-builds/'+generation+'/'+base)
 const env={...process.env,VITE_CREATA_DEMO_MODE:'1',VITE_ADMIN_BASE_PATH:'/'+base,VITE_WEB_URL:'',VITE_EDITORIAL_URL:'/editorial',VITE_HERMES_ADMIN_URL:'/hermes'}
 const child=spawn(process.execPath,['x','vite','build','--configLoader','runner','--base','/'+base+'/','--outDir',output],{cwd:resolve(root,folder),env,stdio:'inherit',windowsHide:true})
 await new Promise<void>((done,fail)=>{child.on('error',fail);child.on('exit',code=>code===0?done():fail(Error(folder+' build failed ('+code+')')))})
 cpSync(output,resolve(root,'creatanews/public',base),{recursive:true})
 mkdirSync(resolve(root,'creatanews/public/demo-apps'),{recursive:true})
 cpSync(resolve(output,'index.html'),resolve(root,'creatanews/public/demo-apps',base+'.html'))
 function scan(folder:string,prefix:string){for(const name of readdirSync(folder)){const path=resolve(folder,name);if(statSync(path).isDirectory())scan(path,prefix+'/'+name);else files.push(prefix+'/'+name)}}
 scan(resolve(output,'assets'),base+'/assets');files.push(base+'/index.html',base+'/creata-mark.svg','demo-apps/'+base+'.html')
}
await build('events-admin','editorial')
await build('agent-admin','hermes')
writeFileSync(resolve(root,'.runtime/demo-admin-assets.json'),JSON.stringify({generation,files},null,2))
console.log('Existing editorial and Hermes demo builds copied into the public frontend')
