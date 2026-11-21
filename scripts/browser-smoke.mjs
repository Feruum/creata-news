import {mkdirSync,writeFileSync} from 'node:fs'
import {resolve} from 'node:path'
import assert from 'node:assert/strict'
process.env.PLAYWRIGHT_BROWSERS_PATH ||= resolve('.runtime/browsers')
const {chromium}=await import('@playwright/test')
const directory=resolve('.runtime/screenshots');mkdirSync(directory,{recursive:true})
const browser=await chromium.launch({headless:true})
const context=await browser.newContext({viewport:{width:1440,height:1000},reducedMotion:'reduce'})
const page=await context.newPage();const errors=[]
page.on('pageerror',error=>errors.push(error.message))
const results=[]
async function visit(url,name){
  const response=await page.goto(url,{waitUntil:'domcontentloaded',timeout:90000})
  assert.equal(response.status(),200,url)
  await page.locator('h1').first().waitFor({timeout:30000})
  await page.evaluate(()=>document.fonts.ready)
  await page.screenshot({path:resolve(directory,name+'.png'),fullPage:true})
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+1)
  assert.equal(overflow,false,'Horizontal overflow: '+url)
  results.push({url,screenshot:name+'.png',title:await page.title()})
}
try{
  await visit('http://localhost:3002/ru','home-desktop')
  await page.locator('[aria-controls="creata-menu"]').click()
  await page.locator('dialog[open]').waitFor()
  await page.screenshot({path:resolve(directory,'menu-desktop.png'),fullPage:true})
  await page.keyboard.press('Escape')
  await page.setViewportSize({width:360,height:800})
  await visit('http://localhost:3002/kk','home-mobile-kk')
  assert.equal(await page.locator('html').getAttribute('lang'),'kk')
  await page.setViewportSize({width:768,height:1000})
  await visit('http://localhost:3002/en','home-tablet-en')
  await page.setViewportSize({width:1440,height:1000})
  const list=await context.request.get('http://localhost:5555/api/creata/v1/public/articles')
  const article=(await list.json()).items[0]
  assert.ok(article,'Explicit development seed required for browser smoke')
  await visit('http://localhost:3002/ru/articles/'+article.slug,'article-desktop')
  assert.ok(await page.locator('h1').textContent())
  assert.ok(await page.locator('link[rel=canonical]').count())
  const login=await context.request.post('http://localhost:5555/api/creata/v1/auth/dev-login',{data:{role:'admin'},headers:{Origin:'http://localhost:6868'}})
  assert.equal(login.status(),200,'Local dev auth must be explicitly enabled')
  await visit('http://localhost:6868/today','editorial-desktop')
  await visit('http://localhost:5174/sources','hermes-desktop')
  await page.setViewportSize({width:768,height:1000})
  await visit('http://localhost:6868/articles','editorial-tablet')
  assert.deepEqual(errors,[],'Browser page errors')
  writeFileSync(resolve('.runtime/browser-results.json'),JSON.stringify({results,errors},null,2))
  console.log(JSON.stringify({screens:results.length,errors,output:directory}))
}finally{await browser.close()}
