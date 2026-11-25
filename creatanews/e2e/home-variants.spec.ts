import {test,expect} from '@playwright/test'

const homeURL=process.env.CREATA_HOME_TEST_URL||process.env.CREATA_WEB_TEST_URL
test.use({baseURL:homeURL,contextOptions:{reducedMotion:'reduce'}})
test.skip(!homeURL,'Requires the local Creata site or an explicit homepage preview')

test('classic is the public home and V2 still works via its direct URL',async({page,request})=>{
 const errors:string[]=[];page.on('pageerror',error=>errors.push(error.message))
 await page.setViewportSize({width:1440,height:1000})
 await page.goto('/ru')
 await expect(page.getByTestId('classic-home-screen')).toBeVisible()
 await expect(page.getByTestId('home-frontpage')).toBeVisible()
 await expect(page.getByTestId('editorial-home-screen')).toHaveCount(0)
 const published=await request.get('/api/creata/v1/public/articles?locale=ru');await expect(published).toBeOK()
 const {items}=await published.json();expect(items.length).toBeGreaterThan(0)
 await expect(page.getByRole('heading',{level:1})).toHaveText(items[0].title.ru)
 await expect(page.locator('.classic-home-utility a[href="/ru/v2"]')).toHaveCount(0)
 await page.goto('/ru/v2')
 await expect(page).toHaveURL(/\/ru\/v2$/)
 await expect(page.getByTestId('editorial-home-screen')).toBeVisible()
 await expect(page.getByTestId('classic-home-screen')).toHaveCount(0)
 await expect(page.getByRole('heading',{level:1})).toHaveText('Будущее начинается там, где ты')
 const portrait=page.locator('.home-intro-art img')
 await expect(portrait).toHaveAttribute('src',/creata-human-hero-v1/)
 await expect.poll(()=>portrait.evaluate(node=>(node as HTMLImageElement).complete&&(node as HTMLImageElement).naturalWidth>0)).toBe(true)
 expect(await page.locator('.site-header .creata-logo-news').evaluate(node=>getComputedStyle(node).fill)).toBe('rgb(75, 68, 246)')
 await page.locator('.site-header .original-menu-trigger').click()
 await page.locator('#creata-menu').getByRole('link',{name:'Главная',exact:true}).click()
 await expect(page).toHaveURL(/\/ru$/)
 await expect(page.getByTestId('classic-home-screen')).toBeVisible()
 expect(errors).toEqual([])
})

test('both homepages fit mobile and V2 locale switching preserves its route',async({page})=>{
 test.setTimeout(90000)
 for(const locale of ['ru','kk','en']){
  for(const suffix of ['','/v2']){
   await page.goto('/'+locale+suffix)
   await expect(page.getByRole('heading',{level:1})).toBeVisible()
   for(const width of [320,360,390,768]){
    await page.setViewportSize({width,height:844})
    await expect.poll(()=>page.evaluate(()=>Math.max(document.body.scrollWidth,document.documentElement.scrollWidth)-innerWidth),{message:`${locale}${suffix} at ${width}px`}).toBeLessThanOrEqual(1)
    const clippedLinks=await page.locator('main .section-heading > a').evaluateAll(links=>links
     .filter(link=>link.getBoundingClientRect().width>0&&link.getBoundingClientRect().right>window.innerWidth+1)
     .map(link=>link.textContent?.trim()))
    expect(clippedLinks,`${locale}${suffix} section links at ${width}px`).toEqual([])
   }
  }
 }
 await page.goto('/ru/v2');await page.setViewportSize({width:390,height:844})
 await page.locator('.site-header .original-menu-trigger').click()
 await page.locator('#creata-menu').getByRole('button',{name:'EN',exact:true}).click()
 await expect(page).toHaveURL(/\/en\/v2$/)
 await expect(page.getByTestId('editorial-home-screen')).toBeVisible()
 await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href',/\/en\/v2$/)
})
