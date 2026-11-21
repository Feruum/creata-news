/** Isolated integration/browser runner. Never targets the working API/database. */
import {spawn,type ChildProcess} from 'node:child_process'
import {resolve} from 'node:path'
import {existsSync,mkdirSync} from 'node:fs'
import {randomBytes} from 'node:crypto'
import {allocatePorts} from './local-ports'
import {startPostizFixture} from './postiz-fixture'
const root=resolve(import.meta.dirname,'..'),dsn=process.env.CREATA_TEST_DATABASE_URL
if(!dsn||!new URL(dsn).pathname.includes('test'))throw Error('CREATA_TEST_DATABASE_URL must name an isolated test database')
const ports=await allocatePorts({CREATA_API_PORT:process.env.CREATA_VERIFY_API_PORT,CREATA_WEB_PORT:process.env.CREATA_VERIFY_WEB_PORT,CREATA_EDITORIAL_PORT:process.env.CREATA_VERIFY_EDITORIAL_PORT,CREATA_HERMES_PORT:process.env.CREATA_VERIFY_HERMES_PORT}),api=`http://127.0.0.1:${ports.api}`,web=`http://127.0.0.1:${ports.web}`,editorial=`http://127.0.0.1:${ports.editorial}`,hermes=`http://127.0.0.1:${ports.hermes}`
const env={...process.env,DATABASE_URL:dsn,CREATA_ENV:'test',CREATA_S3_ENDPOINT:'',CREATA_S3_BUCKET:'',CREATA_S3_ACCESS_KEY:'',CREATA_S3_SECRET_KEY:'',TELEGRAM_BOT_TOKEN:'',CREATA_WORKER_STATE_DIR:resolve(root,'.runtime/verification-hermes-state'),CREATA_DEV_AUTH:'1',CREATA_ALLOW_PRIVATE_SOURCES:'1',CREATA_ADDRESS:`127.0.0.1:${ports.api}`,CREATA_API_URL:api,CREATA_API_ORIGIN:api,CREATA_PUBLIC_URL:web,CREATA_ORIGINS:[web,editorial,hermes].join(','),CREATA_SESSION_SECRET:randomBytes(32).toString('hex'),CREATA_WORKER_TOKEN:randomBytes(32).toString('hex'),CREATA_BOT_SERVICE_TOKEN:randomBytes(32).toString('hex'),CREATA_TELEGRAM_CHANNEL_ID:'-100123456',VITE_WEB_URL:web,VITE_EDITORIAL_URL:editorial,VITE_HERMES_ADMIN_URL:hermes,CREATA_WEB_URL:web,CREATA_WEB_TEST_URL:web,CREATA_EDITORIAL_TEST_URL:editorial,CREATA_HERMES_TEST_URL:hermes,CREATA_TEST_API_ORIGIN:api,CREATA_TEST_BROWSER_ORIGIN:editorial,CREATA_NEWSPAPER_WEB_URL:web,CREATA_MEDIA_DIR:resolve(root,'.runtime/verification-media'),GOCACHE:process.env.GOCACHE||resolve(root,'.runtime/go-cache'),GOMODCACHE:process.env.GOMODCACHE||resolve(root,'.runtime/go-mod')}
mkdirSync(resolve(root,'.runtime/screenshots'),{recursive:true})
const schema='verify_'+randomBytes(8).toString('hex')
let binary=''
const children:ChildProcess[]=[]
const live=new Set<ChildProcess>()
function start(command:string,args:string[],cwd=root){const child=spawn(command,args,{cwd,env,stdio:'inherit',windowsHide:true});children.push(child);live.add(child);child.once('exit',()=>live.delete(child));child.once('error',()=>live.delete(child));return child}
async function command(cmd:string,args:string[],cwd=root){const child=start(cmd,args,cwd);await new Promise<void>((resolve,reject)=>{child.on('error',reject);child.on('exit',code=>code===0?resolve():reject(Error(`${cmd} failed (${code})`)))})}
async function ready(url:string){for(let attempt=0;attempt<180;attempt++){try{if((await fetch(url)).ok)return}catch{}await Bun.sleep(500)}throw Error(`Service not ready: ${url}`)}
async function stop(){const pending=[...live].map(child=>new Promise<void>(resolve=>{child.once('exit',()=>resolve());child.once('error',()=>resolve());child.kill('SIGTERM')}));if(!pending.length)return;await Promise.race([Promise.all(pending),Bun.sleep(5000)]);if(live.size)throw Error('Verification services did not stop; test schema was preserved')}
for(const sig of ['SIGINT','SIGTERM'] as const)process.on(sig,()=>{void stop().finally(()=>process.exit(1))})
const postiz=startPostizFixture(web)
const browserCache = resolve(root, '.runtime/browsers')
Object.assign(env, postiz.env, {
  ...(existsSync(browserCache) && !process.env.PLAYWRIGHT_BROWSERS_PATH && { PLAYWRIGHT_BROWSERS_PATH: browserCache }),
  CREATA_BUILD_DIR: process.env.CREATA_VERIFY_BUILD_DIR || '.next-build',
  NEXT_PUBLIC_CREATA_DEMO_MODE: '0',
  VITE_CREATA_DEMO_MODE: '0',
})
try{
 await command(process.execPath,['scripts/dev-seed.ts'])
 binary=resolve(root,'.runtime',process.platform==='win32'?'creata-verify-api.exe':'creata-verify-api')
 await command('go',['build','-o',binary,'./cmd/server'],resolve(root,'backend/creata'))
 start(binary,['--test-schema',schema,'--seed',resolve(root,'.runtime/dev-seed.json'),'--import-content',resolve(root,'backend/creata/content')]);await ready(api+'/health')
 await command(process.execPath,['scripts/test-live-pipeline.ts'])
 await command(process.execPath,['scripts/test-telegram-pipeline.ts'])
 await command(process.execPath,['run','build'],resolve(root,'creatanews'))
 start(process.execPath,['run','--no-orphans','start','--hostname','127.0.0.1','--port',String(ports.web)],resolve(root,'creatanews'))
 for(const [folder,port]of [['events-admin',ports.editorial],['agent-admin',ports.hermes]] as const)start(process.execPath,['run','--no-orphans','dev','--configLoader','runner','--port',String(port),'--strictPort'],resolve(root,folder))
 await Promise.all([ready(web+'/ru'),ready(editorial),ready(hermes)])
 await command(process.execPath,['x','playwright','test'],resolve(root,'creatanews'))
 await command(process.execPath,['x','playwright','test','--config','playwright.creata.config.ts'],resolve(root,'events-admin'))
 postiz.verify()
 console.log('Full worker/API and public/editorial browser verification passed')
}finally{
 try{
  await stop()
  if(binary)await command(binary,['--drop-test-schema',schema])
 }finally{await postiz.stop()}
}
