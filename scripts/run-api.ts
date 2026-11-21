import {spawn} from 'node:child_process'
import {resolve} from 'node:path'
import {existsSync} from 'node:fs'
import {preserveLocalMedia} from './migrate-local-media'
const root=resolve(import.meta.dirname,'..')
const binary=process.env.CREATA_API_BINARY||resolve(root,'.runtime',process.platform==='win32'?'creata-api.exe':'creata-api')
const args=process.argv.slice(2)
const env={...process.env,GOCACHE:resolve(root,'.runtime/go-cache'),GOMODCACHE:resolve(root,'.runtime/go-mod'),CREATA_CONTENT_IMPORT_DIR:process.env.CREATA_CONTENT_IMPORT_DIR??resolve(root,'backend/creata/content'),CREATA_MEDIA_DIR:resolve(root,process.env.CREATA_MEDIA_DIR||'uploads')}
if(!process.argv.includes('--test-db')&&!env.DATABASE_URL?.includes('test'))preserveLocalMedia(root,env.CREATA_MEDIA_DIR)
if(args.includes('--test-db')){env.DATABASE_URL='postgres://creata:creata-local-test@127.0.0.1:54329/creata_test?sslmode=disable';args.splice(args.indexOf('--test-db'),1)}
if(args.includes('--fixtures')){env.CREATA_ALLOW_PRIVATE_SOURCES='1';args.splice(args.indexOf('--fixtures'),1)}
const source=!process.env.CREATA_API_BINARY||!existsSync(binary)
const command=source?'go':binary
const child=spawn(command,source?['run','./cmd/server',...args]:args,{cwd:source?resolve(root,'backend/creata'):root,env,stdio:'inherit',windowsHide:true})
child.on('exit',code=>process.exit(code||0))
child.on('error',error=>{console.error(error.message);process.exit(1)})
for(const sig of ['SIGINT','SIGTERM'] as const)process.on(sig,()=>child.kill(sig))
