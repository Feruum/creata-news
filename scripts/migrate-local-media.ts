import {existsSync,readdirSync,mkdirSync,readFileSync,copyFileSync} from 'node:fs'
import {createHash} from 'node:crypto'
import {resolve,join,relative,dirname} from 'node:path'

/** Copy prior local API uploads without removing or overwriting an original. */
export function preserveLocalMedia(root:string,target:string){
 let copied=0
 const destination=resolve(target)
 for(const source of [resolve(root,'uploads'),resolve(root,'backend/creata/uploads'),resolve(root,'backend/creata/.runtime/media')]){
  if(source===destination||!existsSync(source))continue
  function visit(directory:string){for(const entry of readdirSync(directory,{withFileTypes:true})){const file=join(directory,entry.name);if(entry.isSymbolicLink())throw Error('A legacy media directory contains a symbolic link');if(entry.isDirectory()){visit(file);continue};const next=join(destination,relative(source,file));if(existsSync(next)){const hash=(p:string)=>createHash('sha256').update(readFileSync(p)).digest('hex');if(hash(file)!==hash(next))throw Error('Legacy media collision: originals preserved; choose a distinct media directory');continue};mkdirSync(dirname(next),{recursive:true});copyFileSync(file,next);copied++}}
  visit(source)
 }if(copied)console.log(`Preserved ${copied} previous API media files in the configured backend store`)
}
