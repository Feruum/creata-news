import {test,expect,type Locator} from '@playwright/test'

test.use({baseURL:process.env.CREATA_HOME_TEST_URL,reducedMotion:'reduce'})
test.skip(!process.env.CREATA_HOME_TEST_URL,'Requires the local Creata preview')

async function assertWordmark(logo:Locator){
 await expect(logo).toBeVisible()
 await expect(logo).toHaveAttribute('aria-label','Creata News')
 await expect(logo.locator('small,.creata-mark-e')).toHaveCount(0)
 const mark=await logo.evaluate(node=>{
  const main=node.querySelector('.creata-logo-name') as SVGGraphicsElement
  const news=node.querySelector('.creata-logo-news') as SVGGraphicsElement
  const style=getComputedStyle(news),canvas=document.createElement('canvas'),context=canvas.getContext('2d')!
  context.font=`${style.fontWeight} ${style.fontSize} ${style.fontFamily}`
  const metrics=context.measureText('NEWS')
  return {nameHeight:main.getBBox().height,newsHeight:metrics.actualBoundingBoxAscent+metrics.actualBoundingBoxDescent,color:getComputedStyle(main).fill,newsColor:style.fill}
 })
 expect(mark.newsHeight/mark.nameHeight).toBeGreaterThan(.8)
 expect(mark.newsHeight/mark.nameHeight).toBeLessThan(1.05)
 expect(mark.color).not.toBe(mark.newsColor)
}

test('reference wordmark fits homepage and internal headers in all locales',async({page})=>{
 test.setTimeout(120000)
 const errors:string[]=[];page.on('pageerror',error=>errors.push(error.message))
 for(const locale of ['ru','kk','en']){
  for(const route of ['','/v2','/latest']){
   await page.goto('/'+locale+route);await page.evaluate(()=>document.fonts.ready)
   const logo=page.locator('.site-header .creata-wordmark')
   await assertWordmark(logo)
   expect(await logo.locator('.creata-logo-name').evaluate(node=>getComputedStyle(node).fill)).toBe('rgb(24, 23, 27)')
   expect(await logo.locator('.creata-logo-news').evaluate(node=>getComputedStyle(node).fill)).toBe('rgb(75, 68, 240)')
   for(const width of [320,390,768,1440]){
    await page.setViewportSize({width,height:1000})
    await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)).toBeLessThanOrEqual(1)
    const positions=await page.locator('.site-header').evaluate(header=>({logo:header.querySelector('.brand-link')!.getBoundingClientRect().right,controls:header.querySelector('.header-controls')!.getBoundingClientRect().left}))
    expect(positions.logo).toBeLessThanOrEqual(positions.controls-2)
   }
  }
 }
 await page.goto('/ru/v2');await page.evaluate(()=>document.fonts.ready)
 await assertWordmark(page.locator('.home-footer .creata-wordmark'))
 await page.locator('.site-header .original-menu-trigger').click()
 await assertWordmark(page.locator('.menu-preferences .creata-wordmark'))
 await page.locator('.home-menu-popular > summary').click()
 await assertWordmark(page.locator('.home-menu-popular .creata-wordmark'))
 expect(errors).toEqual([])
})

test('editorial and Hermes use the full wordmark and the matching compact symbol',async({page})=>{
 await page.setViewportSize({width:1440,height:1000})
 for(const route of ['/editorial/today','/hermes/sources']){
  await page.goto(route)
  const logo=page.locator('.admin-creata-logo')
  await expect(logo).toBeVisible();await page.evaluate(()=>document.fonts.ready)
  await assertWordmark(logo)
  const size=await logo.locator('svg').boundingBox()
  expect(size!.width).toBeGreaterThan(180)
  await page.locator('[data-sidebar="trigger"]').click()
  await expect(logo).not.toBeVisible()
  await expect(page.locator('.admin-creata-mark')).toBeVisible()
  await page.locator('[data-sidebar="trigger"]').click()
  await expect(logo).toBeVisible()
 }
})

test('published newspaper and image fallback retain the reference logo',async({page})=>{
 await page.goto('/ru/gazeta/01')
 const news=page.locator('.newspaper-issue .mh-news').first()
 await expect(news).toBeVisible();await page.evaluate(()=>document.fonts.ready)
 expect(await news.evaluate(node=>getComputedStyle(node).fontFamily)).toContain('Creata Wordmark')
 await page.route('**/brand-test-missing.jpg',route=>route.fulfill({status:404,body:''}))
 await page.goto('/ru/v2')
 const card=page.getByTestId('home-story-rail').locator('article').filter({hasText:'Ucode'})
 await expect.poll(async()=>Number(await page.getByTestId('home-story-rail').locator('..').getAttribute('data-snap-count'))).toBeGreaterThan(1)
 await card.scrollIntoViewIfNeeded()
 await card.locator('img').evaluate(node=>{(node as HTMLImageElement).src='/brand-test-missing.jpg'})
 await assertWordmark(card.locator('.media-fallback .creata-wordmark'))
 const widths=await card.evaluate(node=>({card:node.getBoundingClientRect().width,logo:node.querySelector('.creata-wordmark')!.getBoundingClientRect().width}))
 expect(widths.logo).toBeLessThan(widths.card)
})
