import {createServer} from 'node:net'
export const defaults={api:5555,web:3002,editorial:6868,hermes:5174,bot:3031}
export type LocalPorts=typeof defaults
export async function available(port:number,host:string){return new Promise<boolean>((resolve,reject)=>{const server=createServer();server.once('error',(err:NodeJS.ErrnoException)=>err.code==='EADDRNOTAVAIL'||err.code==='EAFNOSUPPORT'?resolve(true):resolve(false));server.listen({port,host,exclusive:true},()=>server.close(error=>error?reject(error):resolve(true)))})}
export async function allocatePorts(env:Record<string,string|undefined>=process.env):Promise<LocalPorts>{
 const result={...defaults},used=new Set<number>()
 for(const key of Object.keys(result) as Array<keyof LocalPorts>){const value=env[`CREATA_${key.toUpperCase()}_PORT`];let port=value?Number(value):defaults[key];if(!Number.isInteger(port)||port<1024||port>65535)throw Error(`Invalid CREATA_${key.toUpperCase()}_PORT`);const initial=port
  while(used.has(port)||!await available(port,'127.0.0.1')||!await available(port,'::1')){if(value)throw Error(`Configured ${key} port ${port} is occupied. Choose CREATA_${key.toUpperCase()}_PORT; other projects were left running.`);if(++port>initial+100)throw Error(`No free ${key} port near ${initial}`)}result[key]=port;used.add(port)
 }return result
}
