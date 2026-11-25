import {test,expect} from '@playwright/test'

const homeURL=process.env.CREATA_HOME_TEST_URL||process.env.CREATA_DEMO_TEST_URL
test.use({baseURL:homeURL,reducedMotion:'reduce'})
test.skip(!homeURL,'Requires a running public site')

test('public brand fonts and audit fixes hold from mobile to desktop in every locale',async({page})=>{
 test.setTimeout(120000)
 for(const locale of ['ru','kk','en']){
  await page.goto('/'+locale,{waitUntil:'domcontentloaded'})
  await page.evaluate(()=>document.fonts.ready)
  const fonts=await page.evaluate(async()=>{
   const kazakh='ӘәҒғҚқҢңӨөҰұҮүҺһІі ₸'
   const fallbacks=await document.fonts.load('500 16px Montserrat',kazakh)
   return {families:[...document.fonts].map(face=>face.family.replace(/["']/g,'')),body:getComputedStyle(document.body).fontFamily,heading:getComputedStyle(document.querySelector('h1')!).fontFamily,weight:getComputedStyle(document.querySelector('h1')!).fontWeight,kazakhLoaded:fallbacks.length>0&&fallbacks.every(face=>face.status==='loaded')}
  })
  expect(fonts.families).toEqual(expect.arrayContaining(['Unbounded','Involve','Montserrat','Creata Wordmark']))
  expect(fonts.families).not.toContain('Inter');expect(fonts.families).not.toContain('Archivo')
  expect(fonts.body).toContain('Involve');expect(fonts.heading).toContain('Unbounded');expect(fonts.weight).toBe('700');expect(fonts.kazakhLoaded).toBe(true)
  for(const width of [320,390,768,1440]){
   await page.setViewportSize({width,height:844})
   await expect.poll(()=>page.evaluate(()=>Math.max(document.documentElement.scrollWidth,document.body.scrollWidth)-innerWidth)).toBeLessThanOrEqual(1)
   const small=await page.evaluate(()=>[...document.querySelectorAll('body *')].filter(node=>node instanceof HTMLElement&&node.checkVisibility()&&[...node.childNodes].some(child=>child.nodeType===Node.TEXT_NODE&&child.textContent?.trim())&&parseFloat(getComputedStyle(node).fontSize)<12&&node.getBoundingClientRect().width>2&&node.getBoundingClientRect().height>2&&!node.closest('[aria-hidden=true],.sr-only,.skip-link')).map(node=>`${getComputedStyle(node).fontSize} ${node.textContent?.trim().slice(0,35)}`))
   expect(small,`${locale} ${width}px visible text below 12px`).toEqual([])
   expect(await page.getByTestId('home-latest').locator('li:visible').count()).toBe(width<700?4:8)
   await expect(page.locator('header a[href$="/v2"]')).toHaveCount(0)
   const articleImages=page.locator('main .article-card img, [data-testid=home-lead] img, [data-testid=home-secondary] img, .classic-home-feature-band img, [data-opportunity-id] img, [data-component=creata-popular] img')
   for(const image of await articleImages.all())expect(await image.getAttribute('alt')).toBeTruthy()
   if(width<700){
    const touch=await page.evaluate(()=>[...document.querySelectorAll('.site-header button,.site-header a,.bottom-nav a,main button,main a,.classic-home-footer a,.preview-banner a,.preview-banner button')].filter(node=>node instanceof HTMLElement&&node.checkVisibility()&&!node.closest('[aria-hidden=true]')&&node.getBoundingClientRect().width>1&&node.getBoundingClientRect().height>1).map(node=>({label:node.textContent?.trim().slice(0,25)||node.getAttribute('aria-label'),width:node.getBoundingClientRect().width,height:node.getBoundingClientRect().height})).filter(r=>r.width<43.9||r.height<43.9))
    expect(touch,`${locale} ${width}px undersized controls`).toEqual([])
   }
  }
  const popular=page.locator('[data-component=creata-popular]').first()
  await expect(popular).toHaveCSS('background-color','rgb(14, 14, 16)')
  await expect(popular.locator('[class*=rank]').first()).toHaveCSS('color','rgb(75, 68, 246)')
 }
})

test('demo banner can be dismissed from a single compact mobile row',async({page})=>{
 await page.setViewportSize({width:390,height:844});await page.goto('/ru',{waitUntil:'domcontentloaded'})
 const banner=page.getByTestId('demo-notice');test.skip(!await banner.count(),'Demo banner only exists in the demo')
 await expect(banner).toBeVisible();expect((await banner.boundingBox())!.height).toBeLessThanOrEqual(60)
 await banner.getByRole('button',{name:'Закрыть'}).click();await expect(banner).not.toBeVisible()
})

test('newspaper archive and reader use brand headings and interface typography',async({page})=>{
 await page.goto('/ru/gazeta')
 const heading=page.getByRole('heading',{level:1})
 await expect(heading).toHaveCSS('font-weight','700')
 await expect(page.locator('.gazeta-edition h2').first()).toHaveCSS('font-weight','500')
 const tracking=await heading.evaluate(el=>parseFloat(getComputedStyle(el).letterSpacing)/parseFloat(getComputedStyle(el).fontSize))
 expect(tracking).toBeGreaterThanOrEqual(-.026);expect(tracking).toBeLessThanOrEqual(-.019)
 await page.goto('/ru/gazeta/01')
 await expect(page.locator('.desk-bar')).toHaveCSS('font-family',/Montserrat/)
 expect(await page.locator('.desk-bar').evaluate(el=>getComputedStyle(el).fontFamily.split(',')[0])).toContain('Montserrat')
})
