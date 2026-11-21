import {test,expect} from '@playwright/test'

test.use({baseURL:process.env.CREATA_HOME_TEST_URL,reducedMotion:'reduce'})
test.skip(!process.env.CREATA_HOME_TEST_URL,'Requires the local homepage preview')

test('homepage uses distinct article photos while article and listing media stay unchanged',async({page})=>{
 const errors:string[]=[]
 page.on('pageerror',error=>errors.push(error.message))
 await page.setViewportSize({width:1440,height:1000})
 await page.goto('/ru/v2')
 const cards=page.getByTestId('home-story-rail').locator('article')
 await expect(cards).toHaveCount(6)
 const images=cards.locator('img')
 await expect(images).toHaveCount(6)
 const urls=await images.evaluateAll(nodes=>nodes.map(node=>node.getAttribute('src')))
 expect(urls.every(url=>!url?.includes('/maps/'))).toBe(true)
 expect(new Set(urls).size).toBe(6)
 for(const image of await images.all()){
  await image.evaluate(node=>{(node as HTMLImageElement).loading='eager'})
  await expect.poll(()=>image.evaluate(node=>(node as HTMLImageElement).complete&&(node as HTMLImageElement).naturalWidth>0)).toBe(true)
 }
 await expect(cards.filter({hasText:'Ucode'}).locator('img')).toHaveAttribute('src',/ucode-v1\.jpg$/)
 await expect(cards.filter({hasText:'Hong Kong'}).locator('img')).toHaveAttribute('src',/hong-kong-fintech-v2\.jpg$/)
 await page.getByRole('button',{name:'Люди',exact:true}).click()
 await expect(page.getByTestId('home-story-rail').locator('img[src*="/maps/"]')).toHaveCount(0)
 await page.goto('/ru/articles/w-ucode')
 await expect(page.locator('.article-cover img')).toHaveAttribute('src','/demo-media/news/maps/UZ.svg')
 await page.goto('/ru/latest')
 await expect(page.locator('main img[src*="/maps/"]').first()).toBeVisible()
 expect(errors).toEqual([])
})

test('homepage photos work on mobile, in filters and in its popular menu',async({page})=>{
 await page.setViewportSize({width:390,height:844})
 await page.goto('/ru/v2')
 const carousel=page.getByTestId('home-story-rail').locator('..')
 await expect.poll(async()=>Number(await carousel.getAttribute('data-snap-count'))).toBeGreaterThan(1)
 await carousel.focus();await carousel.press('ArrowRight')
 await expect(carousel).toHaveAttribute('data-current-index','1')
 await page.locator('.site-header .original-menu-trigger').click()
 await page.locator('.home-menu-popular > summary').click()
 await expect(page.locator('.home-menu-popular img[src*="/maps/"]')).toHaveCount(0)
 await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)).toBeLessThanOrEqual(1)
})
