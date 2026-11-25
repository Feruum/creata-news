import {test,expect,type Page} from '@playwright/test'
test.use({baseURL:process.env.CREATA_DEMO_TEST_URL,reducedMotion:'reduce'})
test.skip(!process.env.CREATA_DEMO_TEST_URL,'Requires the explicit standalone demo build')
async function fonts(page:Page){return page.evaluate(async()=>{await document.fonts.load('500 16px Inter','ӘҒҚҢӨҰҮҺІ News');await document.fonts.load('700 32px Archivo','Creata News');await document.fonts.ready;return{body:getComputedStyle(document.body).fontFamily,heading:getComputedStyle(document.querySelector('h1')!).fontFamily,inter:document.fonts.check('500 16px Inter'),archivo:document.fonts.check('700 32px Archivo')}})}
test('editorial homepage preserves content and three-column hierarchy in every language',async({page})=>{
 test.setTimeout(120000)
 const errors:string[]=[];page.on('pageerror',error=>errors.push(error.message))
 for(const locale of ['ru','kk','en']){
  await page.setViewportSize({width:1440,height:1100});await page.goto('/'+locale+'/v2')
  await expect(page.getByRole('heading',{level:1})).toHaveCount(1);await expect(page.getByTestId('home-story-rail')).toBeVisible()
  await page.evaluate(()=>document.fonts.ready);const type=await page.evaluate(()=>({body:getComputedStyle(document.body).fontFamily,heading:getComputedStyle(document.querySelector('h1')!).fontFamily,families:[...document.fonts].map(face=>face.family)}));expect(type.body).toContain('Involve');expect(type.heading).toContain('Unbounded');expect(type.families).not.toContain('Inter');expect(type.families).not.toContain('Archivo')
  if(await page.locator('html').getAttribute('data-theme')!=='light'){await page.locator('.site-header .original-menu-trigger').click();await page.locator('#creata-menu .theme-toggle').click();await page.keyboard.press('Escape')}
  await expect(page.locator('.home-header-nav')).toBeVisible()
  const lead=await page.getByTestId('home-lead').boundingBox(),rail=await page.getByTestId('home-story-rail').boundingBox()
  expect(rail!.y).toBeGreaterThan(lead!.y+lead!.height)
  const columns=await page.getByTestId('home-story-rail').locator('article').evaluateAll(nodes=>nodes.slice(0,3).map(node=>node.getBoundingClientRect().y))
  expect(columns).toHaveLength(3);expect(Math.max(...columns)-Math.min(...columns)).toBeLessThan(2)
  await expect(page.getByTestId('home-newspaper-promo')).toBeVisible()
  await page.getByTestId('home-newspaper-promo').scrollIntoViewIfNeeded()
  await expect.poll(()=>page.locator('.home-newspaper-cover img').evaluate(e=>(e as HTMLImageElement).complete&&(e as HTMLImageElement).naturalWidth>0),{timeout:15000}).toBe(true)
  expect((await page.getByTestId('home-newspaper-promo').boundingBox())!.height).toBeLessThan(500)
  expect((await page.locator('.home-footer .creata-logo-full').boundingBox())!.width,'Approved footer uses the large original wordmark').toBeGreaterThan(1200)
  const footer=await page.locator('.home-footer').boundingBox();expect(footer!.height).toBeLessThan(750);expect(Math.abs(footer!.x)).toBeLessThan(1);expect(Math.abs(footer!.width-1440)).toBeLessThan(1)
  for(const width of [360,768,1000]){
   await page.setViewportSize({width,height:1000})
   expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false)
   await expect.poll(()=>page.locator('.home-footer-brand').evaluate(e=>e.scrollWidth-e.clientWidth),{message:`Footer fits ${locale} at ${width}px`}).toBeLessThanOrEqual(1)
   await expect.poll(()=>page.evaluate(()=>Math.abs(document.querySelector('.home-footer-grid')!.getBoundingClientRect().left-document.querySelector('.app-shell')!.getBoundingClientRect().left))).toBeLessThan(1)
  }
 }
 expect(errors).toEqual([])
})
test('Inter and Archivo are local in both admin apps and home carousel still wraps',async({page})=>{
 await page.setViewportSize({width:1440,height:1000})
 for(const path of ['/editorial/today','/hermes/sources']){await page.goto(path);await expect(page.getByRole('heading',{level:1}).first()).toBeVisible();const type=await fonts(page);expect(type.body).toContain('Inter');expect(type.heading).toContain('Archivo');expect(type.inter&&type.archivo).toBe(true)}
 await page.setViewportSize({width:390,height:844});await page.goto('/ru/v2');const track=page.getByTestId('home-story-rail'),viewport=track.locator('..'),section=track.locator('xpath=ancestor::section[1]');await viewport.scrollIntoViewIfNeeded();await expect.poll(async()=>Number(await viewport.getAttribute('data-snap-count'))).toBeGreaterThan(2)
 const count=Number(await viewport.getAttribute('data-snap-count'));await section.getByRole('button',{name:/Предыдущий материал:/}).click();await expect(viewport).toHaveAttribute('data-current-index',String(count-1));await section.getByRole('button',{name:/Следующий материал:/}).click();await expect(viewport).toHaveAttribute('data-current-index','0');await viewport.focus();await viewport.press('ArrowRight');await expect(viewport).toHaveAttribute('data-current-index','1');await viewport.hover();await page.mouse.wheel(-120,0);await expect(viewport).toHaveAttribute('data-current-index','0')
 const previousTheme=await page.locator('html').getAttribute('data-theme');await page.locator('.site-header .original-menu-trigger').click();await page.locator('#creata-menu .theme-toggle').click();await expect(page.locator('html')).toHaveAttribute('data-theme',previousTheme==='dark'?'light':'dark');expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false)
})
test('homepage newspaper promotion follows the published CMS issue',async({page,context})=>{
 const publicPage=await context.newPage();await publicPage.goto('/ru/v2');const promo=publicPage.getByTestId('home-newspaper-promo');await expect(promo).toBeVisible();await expect(promo.getByRole('link',{name:'Читать выпуск',exact:true})).toHaveAttribute('href','/ru/gazeta/01')
 await page.goto('/editorial/content');await page.getByRole('button',{name:'Газета №01',exact:true}).click();await page.getByRole('button',{name:'Снять с публикации',exact:true}).click();await expect(promo).toHaveCount(0)
 await expect(page.getByRole('button',{name:'Опубликовать',exact:true})).toBeEnabled();await page.getByRole('button',{name:'Опубликовать',exact:true}).click();await expect(promo).toBeVisible()
})
