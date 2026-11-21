import {test,expect,type Page} from '@playwright/test'

test.use({baseURL:process.env.CREATA_HOME_TEST_URL,viewport:{width:1440,height:1000},reducedMotion:'no-preference'})
test.skip(!process.env.CREATA_HOME_TEST_URL,'Requires the local homepage preview')

async function recordMotion(page:Page){
 await page.addInitScript(()=>{
  const calls:{owner:string;duration:number;properties:string[]}[]=[]
  Object.assign(window,{homeMotionCalls:calls})
  const original=Element.prototype.animate
  Element.prototype.animate=function(keyframes,options){
   const owner=this.matches('[aria-roledescription="carousel"]')&&this.querySelector('[data-testid="home-story-rail"]')?'filter':this.classList.contains('home-newspaper-copy')?'newspaper':''
   if(owner)calls.push({owner,duration:Number(typeof options==='number'?options:options?.duration||0),properties:Array.isArray(keyframes)?Object.keys(keyframes[0]||{}):Object.keys(keyframes||{})})
   return original.call(this,keyframes,options)
  }
 })
}
const motionCalls=(page:Page)=>page.evaluate(()=>(window as unknown as {homeMotionCalls:{owner:string;duration:number;properties:string[]}[]}).homeMotionCalls)

test('homepage motion gives pointer feedback and reveals a section only once',async({page})=>{
 await recordMotion(page)
 await page.goto('/ru/v2')
 await expect(page.locator('.home-intro-title')).toBeVisible()
 expect(await page.locator('.home-intro-title').evaluate(node=>getComputedStyle(node).animationName)).not.toBe('none')
 const card=page.getByTestId('home-story-rail').locator('article').first()
 const image=card.locator('.media-frame img')
 await card.hover()
 await expect.poll(()=>image.evaluate(node=>new DOMMatrixReadOnly(getComputedStyle(node).transform).a)).toBeGreaterThan(1.01)
 await page.getByRole('button',{name:'Технологии',exact:true}).click()
 await expect.poll(async()=>(await motionCalls(page)).filter(call=>call.owner==='filter').length).toBeGreaterThan(0)
 const filter=(await motionCalls(page)).find(call=>call.owner==='filter')!
 expect(filter.duration).toBeLessThanOrEqual(200)
 expect(filter.properties).toEqual(['opacity'])
 await page.getByRole('button',{name:'Все материалы',exact:true}).click()
 await expect.poll(()=>page.getByTestId('home-story-rail').locator('..').evaluate(node=>getComputedStyle(node).opacity)).toBe('1')
 const promo=page.locator('.home-newspaper-copy')
 await promo.scrollIntoViewIfNeeded()
 await expect.poll(async()=>(await motionCalls(page)).filter(call=>call.owner==='newspaper').length).toBeGreaterThan(0)
 const firstReveal=(await motionCalls(page)).filter(call=>call.owner==='newspaper')
 expect([...new Set(firstReveal.flatMap(call=>call.properties))].sort()).toEqual(['opacity','transform'])
 await page.locator('.site-header').scrollIntoViewIfNeeded()
 await promo.scrollIntoViewIfNeeded()
 expect((await motionCalls(page)).filter(call=>call.owner==='newspaper')).toHaveLength(firstReveal.length)
 await expect.poll(()=>promo.evaluate(node=>getComputedStyle(node).opacity)).toBe('1')
 await page.screenshot({path:test.info().outputPath('motion-desktop.png'),fullPage:true})
})

test('keyboard filters remain immediate and rapid changes keep the final selection',async({page})=>{
 await recordMotion(page);await page.goto('/ru/v2')
 const science=page.getByRole('button',{name:'Наука',exact:true})
 await science.focus();await science.press('Enter')
 await expect(science).toHaveAttribute('aria-pressed','true')
 expect((await motionCalls(page)).filter(call=>call.owner==='filter')).toHaveLength(0)
 for(const label of ['Технологии','Люди','Все материалы'])await page.getByRole('button',{name:label,exact:true}).click()
 await expect(page.getByRole('button',{name:'Все материалы',exact:true})).toHaveAttribute('aria-pressed','true')
 await expect(page.getByTestId('home-story-rail').locator('article')).toHaveCount(6)
 await expect.poll(()=>page.getByTestId('home-story-rail').locator('..').evaluate(node=>getComputedStyle(node).opacity)).toBe('1')
})

test('reduced motion keeps reading and interactions free of movement',async({page})=>{
 await page.emulateMedia({reducedMotion:'reduce'})
 await recordMotion(page);await page.goto('/ru/v2')
 expect(await page.locator('.home-intro-title').evaluate(node=>getComputedStyle(node).animationName)).toBe('none')
 const card=page.getByTestId('home-story-rail').locator('article').first()
 await card.hover()
 expect(await card.locator('.media-frame img').evaluate(node=>getComputedStyle(node).transform)).toBe('none')
 await page.getByRole('button',{name:'Технологии',exact:true}).click()
 await page.locator('.home-newspaper-copy').scrollIntoViewIfNeeded()
 expect(await motionCalls(page)).toEqual([])
 await expect(page.locator('#home-newspaper-title')).toBeVisible()
})
