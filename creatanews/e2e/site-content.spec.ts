import {test,expect} from '@playwright/test'
import {resolve,dirname} from 'node:path'

test('ecosystem and about use published backend documents and retain their responsive Creata components',async({page,request},info)=>{
 test.setTimeout(120000)
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message))
 const result=await request.get('/api/creata/v1/public/pages/ecosystem');expect(result.ok()).toBeTruthy()
 const content=(await result.json()).content
 for(const locale of ['ru','kk','en']){
  await page.goto(`/${locale}/ecosystem`)
  await expect(page.locator('#ecosystem-title')).toHaveText(content.copy[locale].hero.join(''))
  await expect(page.locator('[data-product-id]')).toHaveCount(content.products.length)
  expect(await page.locator('[data-testid="ecosystem-page"] img').evaluateAll(images=>images.every(img=>(img as HTMLImageElement).src.includes('/media/imports/')))).toBe(true)
 }
 for(const width of [360,768,1440]){await page.setViewportSize({width,height:1000});await page.goto('/ru/ecosystem');expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true)}
 await page.emulateMedia({reducedMotion:'reduce'})
 await page.screenshot({path:resolve(dirname(info.config.configFile!),'../.runtime/screenshots/ecosystem-backend-desktop.png'),fullPage:true})
 await page.goto('/ru/about')
 const about=(await(await request.get('/api/creata/v1/public/pages/about')).json()).content.ru
 await expect(page.getByRole('heading',{level:1})).toHaveText(about.title)
 expect(errors).toEqual([])
})

test('original image URLs are forwarded to backend media storage',async({request})=>{
 const image=await request.get('/newspaper/issues/01/cover.png',{maxRedirects:0})
 expect(image.status()).toBe(307)
 expect(image.headers().location).toContain('/media/imports/')
})

test('reduced-motion cards hydrate without warnings and are visible immediately',async({page})=>{
 const errors:string[]=[];page.on('console',message=>{if(message.type()==='error'&&/hydrat/i.test(message.text()))errors.push(message.text())});page.on('pageerror',error=>errors.push(error.message))
 await page.emulateMedia({reducedMotion:'reduce'})
 await page.goto('/ru');const card=page.getByTestId('opportunities-band').locator('article').first();await card.waitFor()
 expect(await card.evaluate(element=>({opacity:getComputedStyle(element).opacity,transform:getComputedStyle(element).transform}))).toEqual({opacity:'1',transform:'none'})
 await card.scrollIntoViewIfNeeded();expect(errors).toEqual([])
})
