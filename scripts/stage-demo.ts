/** Curated frontend-only deployment. Never stages the repository root or .env. */
import {cpSync,mkdirSync,mkdtempSync,readFileSync,writeFileSync,readdirSync,statSync,existsSync} from 'node:fs'
import {resolve,relative,basename,sep} from 'node:path'
import {prepareDemoAssets} from './prepare-demo-assets'
const root=resolve(import.meta.dirname,'..'),deploy=resolve(root,'.runtime/deploy')
mkdirSync(deploy,{recursive:true})
const stage=mkdtempSync(resolve(deploy,'creata-news-demo-'))
const excluded=new Set(['node_modules','.git','.codex','.agents','.vercel','dist','test-results','e2e','coverage'])
function include(path:string){const parts=relative(root,path).split(sep);return!parts.some(part=>excluded.has(part)||part.startsWith('.env')||part.startsWith('.next'))&&!/\.(?:test|spec)\.[tj]sx?$|\.tsbuildinfo$/.test(path)}
function copy(from:string,to=from){cpSync(resolve(root,from),resolve(stage,to),{recursive:true,filter:include})}
copy('creatanews/src','src');copy('creatanews/next.config.ts','next.config.ts');copy('creatanews/tsconfig.json','tsconfig.json')
for(const name of ['brand','contracts','graphics','demo-pipeline'])copy('packages/'+name)
for(const folder of ['fonts','newspaper/fonts'])copy('creatanews/public/'+folder,'public/'+folder)
for(const asset of ['creata-human-hero-v1.png','creata-maker-v1.png','stories'])copy('creatanews/public/images/home/'+asset,'public/images/home/'+asset)
await prepareDemoAssets(resolve(stage,'public'))
const pkg=JSON.parse(readFileSync(resolve(root,'creatanews/package.json'),'utf8'))
pkg.workspaces=['packages/*'];pkg.packageManager=JSON.parse(readFileSync(resolve(root,'package.json'),'utf8')).packageManager
pkg.engines={node:'24.x'}
// Deploy the exact dependency versions already built and checked locally.
for(const group of ['dependencies','devDependencies'])for(const [name,range]of Object.entries(pkg[group]||{})){
 if(String(range).startsWith('workspace:'))continue
 const file=[resolve(root,'creatanews/node_modules',name,'package.json'),resolve(root,'node_modules',name,'package.json')].find(path=>existsSync(path))
 if(!file)throw Error('Cannot resolve tested dependency '+name)
 pkg[group][name]=JSON.parse(readFileSync(file,'utf8')).version
}
pkg.scripts={build:'next build',start:'next start',typecheck:'next typegen && tsc --noEmit'}
writeFileSync(resolve(stage,'package.json'),JSON.stringify(pkg,null,2))
writeFileSync(resolve(stage,'vercel.json'),JSON.stringify({framework:'nextjs',installCommand:'bun install --frozen-lockfile',buildCommand:'bun run build'},null,2))
writeFileSync(resolve(stage,'.vercelignore'),'node_modules\n.next*\n.env*\n*.tsbuildinfo\n')
const files:{path:string;bytes:number}[]=[]
function scan(folder:string){for(const name of readdirSync(folder)){const path=resolve(folder,name),stat=statSync(path);if(stat.isDirectory())scan(path);else{if(basename(path).startsWith('.env'))throw Error('A secret file entered the staging allowlist');files.push({path:relative(stage,path).split(sep).join('/'),bytes:stat.size})}}}
scan(stage)
const manifest={stage,files:files.length,bytes:files.reduce((sum,file)=>sum+file.bytes,0),included:files}
writeFileSync(resolve(root,'.runtime/vercel-demo-stage.json'),JSON.stringify(manifest,null,2))
console.log(JSON.stringify({stage,files:manifest.files,bytes:manifest.bytes,backendIncluded:false,envFilesIncluded:false}))
