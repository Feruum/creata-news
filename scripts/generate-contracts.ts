import {readFileSync,writeFileSync,existsSync} from 'node:fs'
const input=new URL('../backend/creata/openapi.json',import.meta.url)
const output=new URL('../packages/contracts/src/generated.ts',import.meta.url)
const doc=JSON.parse(readFileSync(input,'utf8'))
function type(s:any):string {
  if(!s)return 'unknown'
  if(s.$ref)return `schemas[${JSON.stringify(s.$ref.split('/').pop())}]`
  if(s.enum)return s.enum.map((v:unknown)=>JSON.stringify(v)).join(' | ')
  if(s.oneOf||s.anyOf)return (s.oneOf||s.anyOf).map(type).join(' | ')
  if(s.allOf)return s.allOf.map(type).join(' & ')
  if(s.type==='array')return `Array<${type(s.items)}>`
  if(s.type==='object'||s.properties){
    const fields=Object.entries(s.properties||{}).map(([key,value])=>`${JSON.stringify(key)}${s.required?.includes(key)?'':'?'}: ${type(value)};`)
    if(s.additionalProperties)fields.push(`[key: string]: ${s.additionalProperties===true?'unknown':type(s.additionalProperties)};`)
    return `{ ${fields.join(' ')} }`
  }
  return ({integer:'number',number:'number',boolean:'boolean',string:'string',null:'null'} as Record<string,string>)[s.type]||'unknown'
}
let result='// Generated from backend/creata/openapi.json. Run bun run contracts.\nexport interface schemas {\n'
for(const [name,schema] of Object.entries(doc.components?.schemas||{}))result+=`  ${JSON.stringify(name)}: ${type(schema)}\n`
result+='}\nexport interface endpoints {\n'
for(const [path,methods] of Object.entries(doc.paths||{}))for(const [method,operation] of Object.entries(methods as object)){
  const op=operation as any
  const response=op.responses?.['200']?.content?.['application/json']?.schema||op.responses?.['201']?.content?.['application/json']?.schema
  result+=`  ${JSON.stringify(method.toUpperCase()+' '+path)}: { request: ${type(op.requestBody?.content?.['application/json']?.schema)}; response: ${type(response)} }\n`
}
result+='}\n'
if(process.argv.includes('--check')){if(!existsSync(output)||readFileSync(output,'utf8')!==result){console.error('Generated contracts are stale');process.exit(1)}}
else {writeFileSync(output,result);console.log(`Generated ${Object.keys(doc.components?.schemas||{}).length} schemas`)}
