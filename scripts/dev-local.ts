import {spawn,type ChildProcess} from 'node:child_process'
import {mkdirSync,writeFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {allocatePorts} from './local-ports'
import {preserveLocalMedia} from './migrate-local-media'
import {ensureLocalDatabase} from './local-database'
import {resolveHermesPython} from './local-hermes'
const root=resolve(import.meta.dirname,'..'),ports=await allocatePorts(),host='127.0.0.1'
const urls={api:`http://${host}:${ports.api}`,web:`http://localhost:${ports.web}`,editorial:`http://localhost:${ports.editorial}`,hermes:`http://localhost:${ports.hermes}`,bot:`http://${host}:${ports.bot}`}
const env={...process.env,CREATA_ADDRESS:`${host}:${ports.api}`,CREATA_API_URL:`http://localhost:${ports.api}`,CREATA_API_ORIGIN:urls.api,CREATA_PUBLIC_URL:urls.web,CREATA_EDITORIAL_URL:urls.editorial,CREATA_HERMES_ADMIN_URL:urls.hermes,CREATA_ORIGINS:[urls.web,urls.editorial,urls.hermes,...[ports.web,ports.editorial,ports.hermes].map(p=>`http://${host}:${p}`)].join(','),VITE_WEB_URL:urls.web,VITE_EDITORIAL_URL:urls.editorial,VITE_HERMES_ADMIN_URL:urls.hermes,HTTP_PORT:String(ports.bot),GOCACHE:resolve(root,'.runtime/go-cache'),GOMODCACHE:resolve(root,'.runtime/go-mod'),CREATA_CONTENT_IMPORT_DIR:resolve(root,'backend/creata/content'),CREATA_MEDIA_DIR:resolve(root,process.env.CREATA_MEDIA_DIR||'uploads'),CREATA_WORKER_STATE_DIR:resolve(root,process.env.CREATA_WORKER_STATE_DIR||'.runtime/hermes-state')}
console.log('Creata local URLs:',urls)
if(process.argv.includes('--check'))process.exit(0)
await ensureLocalDatabase(process.env.DATABASE_URL)
const python=resolveHermesPython()
mkdirSync(resolve(root,'.runtime'),{recursive:true})
preserveLocalMedia(root,env.CREATA_MEDIA_DIR)
const children:ChildProcess[]=[];let stopping=false
const serviceEnv={...env,CREATA_POSTIZ_API_KEY:''}
function start(cmd:string,args:string[],cwd=root,childEnv:NodeJS.ProcessEnv=serviceEnv){const child=spawn(cmd,args,{cwd,env:childEnv,stdio:'inherit',windowsHide:true});children.push(child);child.on('error',err=>{console.error(err.message);stop(1)});child.on('exit',code=>{if(!stopping){console.error(`${cmd} exited (${code})`);stop(code??1)}});return child}
function stop(code=0){if(stopping)return;stopping=true;for(const child of children)child.kill('SIGTERM');process.exitCode=code}
for(const sig of ['SIGINT','SIGTERM'] as const)process.on(sig,()=>stop())
// Build once and launch the binary directly so shutdown also stops the API on Windows.
const binary=resolve(root,'.runtime',process.platform==='win32'?'creata-local-api.exe':'creata-local-api')
const build=spawn('go',['build','-o',binary,'./cmd/server'],{cwd:resolve(root,'backend/creata'),env:serviceEnv,stdio:'inherit',windowsHide:true})
await new Promise<void>((resolve,reject)=>{build.on('error',reject);build.on('exit',code=>code===0?resolve():reject(Error('API build failed')))})
start(binary,[],root,env)
let ready=false;for(let attempt=0;attempt<120&&!stopping;attempt++){try{ready=(await fetch(urls.api+'/health')).ok}catch{}if(ready)break;await Bun.sleep(500)}
if(!ready){stop(1);throw Error('API was not ready; inspect database/import diagnostics above')}
start(process.execPath,['run','--no-orphans','dev','--hostname','127.0.0.1','--port',String(ports.web)],resolve(root,'creatanews'))
start(process.execPath,['run','--no-orphans','dev','--configLoader','runner','--host',host,'--port',String(ports.editorial),'--strictPort'],resolve(root,'events-admin'))
start(process.execPath,['run','--no-orphans','dev','--configLoader','runner','--host',host,'--port',String(ports.hermes),'--strictPort'],resolve(root,'agent-admin'))
start(process.execPath,['run','--no-orphans','dev'],resolve(root,'telegram-bot'))
start(python,['-m','hermes.main'],resolve(root,'hermes-worker'))
writeFileSync(resolve(root,'.runtime/local-ports.json'),JSON.stringify({ports,urls},null,2)+'\n')
