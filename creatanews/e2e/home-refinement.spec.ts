import {test,expect} from '@playwright/test'

test.use({baseURL:process.env.CREATA_HOME_TEST_URL,reducedMotion:'reduce',viewport:{width:390,height:844}})
test.skip(!process.env.CREATA_HOME_TEST_URL,'Requires the local homepage preview')

test('homepage removes overlays while keeping discovery and preferences in its menu',async({page})=>{
 await page.goto('/ru/v2')
 await expect(page.locator('.home-shell .bottom-nav')).toHaveCount(0)
 await expect(page.locator('.site-header .theme-toggle')).toHaveCount(0)
 await expect(page.locator('main .home-more-stories')).toHaveCount(0)
 await expect(page.locator('main').getByText('Возможности рядом',{exact:true})).toHaveCount(0)
 await page.locator('.site-header .original-menu-trigger').click()
 const menu=page.locator('#creata-menu')
 await expect(menu.getByRole('button',{name:'EN',exact:true})).toBeVisible()
 const theme=await page.locator('html').getAttribute('data-theme')
 await menu.locator('.theme-toggle').click()
 await expect(page.locator('html')).toHaveAttribute('data-theme',theme==='light'?'dark':'light')
 await menu.locator('.theme-toggle').click()
 const popular=menu.locator('.home-menu-popular')
 await popular.locator('summary').click()
 const carousel=popular.locator('[aria-roledescription="carousel"]')
 await expect.poll(async()=>Number(await carousel.getAttribute('data-snap-count'))).toBeGreaterThan(1)
 await carousel.focus();await carousel.press('ArrowRight')
 await expect(carousel).toHaveAttribute('data-current-index','1')
 await menu.getByRole('link',{name:'Возможности',exact:true}).click()
 await expect(page).toHaveURL(/\/ru\/opportunities$/)
 await expect(page.getByRole('heading',{level:1})).toBeVisible()
})

test('homepage refinement keeps the latest campaign and leaves adjacent pages intact',async({page})=>{
 await page.goto('/ru/v2')
 const paragraph=page.getByTestId('home-story-rail').locator('.card-lead').first()
 expect(await paragraph.evaluate(node=>parseFloat(getComputedStyle(node).fontSize))).toBeGreaterThanOrEqual(14)
 expect(await page.getByTestId('home-story-rail').locator('.card-meta').first().evaluate(node=>parseFloat(getComputedStyle(node).fontSize))).toBeGreaterThanOrEqual(11)
 await expect(page.locator('.home-feature-band img')).toHaveAttribute('src',/creata-maker-v1/)
 expect(await page.getByTestId('home-newspaper-promo').evaluate(node=>getComputedStyle(node).backgroundImage)).toBe('none')
 await expect(page.getByTestId('home-newspaper-promo').getByRole('link',{name:'Читать выпуск',exact:true})).toHaveAttribute('href',/\/ru\/gazeta\/.+/)
 for(const route of ['/ru/about','/ru/latest']){
  await page.goto(route)
  await expect(page.getByRole('heading',{level:1}).first()).toBeVisible()
  await expect(page.locator('.site-header .theme-toggle')).toBeVisible()
  await expect(page.locator('.bottom-nav')).toBeVisible()
 }
})

test('homepage carousels keep the first card aligned after opening and resizing',async({page})=>{
 await page.goto('/ru/v2')
 const latest=page.getByTestId('home-story-rail').locator('..')
 await expect.poll(async()=>Number(await latest.getAttribute('data-snap-count'))).toBeGreaterThan(1)
 for(const width of [360,390]){
  await page.setViewportSize({width,height:844})
  await page.evaluate(()=>new Promise<void>(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve()))))
  await expect.poll(()=>latest.evaluate(root=>Math.abs(root.querySelector('article')!.getBoundingClientRect().left-root.getBoundingClientRect().left))).toBeLessThan(1)
 }
 await page.locator('.site-header .original-menu-trigger').click()
 await page.locator('.home-menu-popular > summary').click()
 const popular=page.locator('.home-menu-popular [aria-roledescription="carousel"]')
 await expect.poll(async()=>Number(await popular.getAttribute('data-snap-count'))).toBeGreaterThan(1)
 await popular.scrollIntoViewIfNeeded()
 await expect.poll(()=>popular.evaluate(root=>(root.firstElementChild as HTMLElement).offsetLeft)).toBe(0)
 for(const width of [360,390]){
  await page.setViewportSize({width,height:844})
  await page.evaluate(()=>new Promise<void>(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve()))))
  await expect.poll(()=>popular.evaluate(root=>Math.abs(root.querySelector('li')!.getBoundingClientRect().left-root.getBoundingClientRect().left))).toBeLessThan(1)
 }
})
