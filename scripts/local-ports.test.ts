import {test,expect} from 'bun:test'
import {createServer} from 'node:net'
import {allocatePorts} from './local-ports'
test('explicit occupied port is rejected without touching the listener',async()=>{
 const server=createServer();await new Promise<void>(resolve=>server.listen(0,'127.0.0.1',resolve));try{const address=server.address();if(!address||typeof address==='string')throw Error('missing address');await expect(allocatePorts({CREATA_API_PORT:String(address.port)})).rejects.toThrow('is occupied');expect(server.listening).toBe(true)}finally{await new Promise<void>(resolve=>server.close(()=>resolve()))}
})
