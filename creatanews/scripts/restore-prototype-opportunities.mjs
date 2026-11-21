/** Reproduce the original opportunity CSS extraction, without adding Tailwind
 * globally or changing the existing frontend. Run with Bun from creatanews/. */
import fs from 'node:fs'
import { createRequire } from 'node:module'
import { createHash } from 'node:crypto'
const postcss=createRequire(import.meta.resolve('next/package.json'))('postcss')
const source='../creata-news-v2/',destination='src/features/opportunities/'
const home=fs.readFileSync(source+'source-v2/src/home.jsx','utf8')
const ui=fs.readFileSync(source+'source-v2/src/ui.jsx','utf8')
const card=home.slice(home.indexOf('export function OppCard('),home.indexOf('\nfunction OppsBand('))
const band=home.slice(home.indexOf('function OppsBand('),home.indexOf('\nfunction IssueBand('))
const countdown=ui.slice(ui.indexOf('export function Countdown('),ui.indexOf('/* ---------- След из фото'))
const originalCss=fs.readFileSync(source+'site-v2/assets/app.css','utf8')
const classes=new Set([...`${card}\n${countdown}\n${band}`.matchAll(/"([^"]*)"/g)].flatMap(match=>match[1].split(/\s+/)).filter(Boolean))
classes.delete('group')
const selectors=[...classes].map(value=>'.'+value.replace(/[^\w-]/g,'\\$&'))
const css=postcss.parse(originalCss)
const literal=value=>value.replace(/[.*+?^${}()|[\]\\]/g,'\\$&')
function select(container){
  for(const node of [...container.nodes||[]]){
    if(node.type==='atrule'){select(node);if(!node.nodes?.length)node.remove()}
    else if(node.type==='rule'){
      if(!selectors.some(selector=>new RegExp(literal(selector)+'(?=[:\\s,#>+~.)\\[]|$)').test(node.selector)))node.remove()
      else node.selector=node.selector.split(',').map(selector=>'.prototype-opportunity-scope '+selector).join(',')
    }else node.remove()
  }
}
select(css)
const colors={paper:'page',band:'surface',card:'panel',ink:'text',muted:'muted',rule:'border',accent:'accent','accent-ink':'accent',soft:'soft',onaccent:'on-accent'}
css.walkDecls(declaration=>{
  let value=declaration.value
  for(const [original,shared]of Object.entries(colors)){
    const prefix=`rgb(var(--${original})`;let start
    while((start=value.indexOf(prefix))>=0){
      let end=start+prefix.length,depth=1
      for(;end<value.length;end++){if(value[end]==='(')depth++;if(value[end]===')'&&!--depth)break}
      const rest=value.slice(start+prefix.length,end).trim(),alpha=rest.startsWith('/')?rest.slice(1).trim():''
      const replacement=alpha?`color-mix(in srgb,var(--creata-${shared}) calc(100% * ${alpha}),transparent)`:`var(--creata-${shared})`
      value=value.slice(0,start)+replacement+value.slice(end+1)
    }
  }
  declaration.value=value
})
fs.mkdirSync(destination+'reference',{recursive:true})
fs.writeFileSync(destination+'reference/OppCard.jsx.txt',card)
fs.writeFileSync(destination+'reference/Countdown.jsx.txt',countdown)
fs.writeFileSync(destination+'reference/OppsBand.jsx.txt',band)
fs.writeFileSync(destination+'prototype-card.css','/* Original opportunity utilities extracted from creata-news-v2/site-v2/assets/app.css. Selectors are scoped; semantic colors consume shared Creata tokens. */\n.prototype-opportunity-scope{min-width:0;height:100%}\n.prototype-opportunity-scope,.prototype-opportunity-scope *{--tw-ring-inset:;--tw-ring-offset-width:0px;--tw-ring-offset-color:#fff;--tw-ring-color:#4b44f0;--tw-ring-offset-shadow:0 0 #0000;--tw-ring-shadow:0 0 #0000;--tw-shadow:0 0 #0000;--tw-shadow-colored:0 0 #0000;--tw-scale-x:1;--tw-scale-y:1;--tw-translate-x:0;--tw-translate-y:0;--tw-rotate:0;--tw-skew-x:0;--tw-skew-y:0}\n'+css.toString()+'\n')
const sha=value=>createHash('sha256').update(value).digest('hex')
fs.writeFileSync('PROTOTYPE-SOURCE-REUSE.json',JSON.stringify([
  {original:'creata-news-v2/source-v2/src/home.jsx#OppCard',sourceSha256:sha(card),preserved:destination+'reference/OppCard.jsx.txt',active:destination+'opportunity-card.tsx',reuse:'Original OppCard markup and Tailwind classes; API and locale props replace CN hooks; persisted Save replaces local reminder.'},
  {original:'creata-news-v2/source-v2/src/ui.jsx#Countdown',sourceSha256:sha(countdown),preserved:destination+'reference/Countdown.jsx.txt',active:destination+'countdown.tsx',reuse:'Original digit grid and FlowDigit; hydration-safe timer, explicit locale and reduced motion.'},
  {original:'creata-news-v2/source-v2/src/home.jsx#OppsBand',sourceSha256:sha(band),preserved:destination+'reference/OppsBand.jsx.txt',active:destination+'opportunities-band.tsx',reuse:'Original band header and lead, original OppCard; static grid extended to a 4/2/1-card looping carousel per user request. API data replaces CN.liveOpps, fixed colors adapt to current Creata theme.'},
  {original:'creata-news-v2/site-v2/assets/app.css',sourceSha256:sha(originalCss),active:destination+'prototype-card.css',reuse:'Original compiled utility declarations for OppCard and Countdown; scoped and semantic colors mapped to @creata/brand.'}
],null,2)+'\n')
console.log(`Restored ${css.nodes.length} original CSS rules and ${classes.size} card classes`)
