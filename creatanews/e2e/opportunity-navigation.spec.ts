import { expect,test,type Page } from '@playwright/test'

async function load(page:Page){
  const hydrated=page.waitForResponse(response=>response.url().endsWith('/auth/me'))
  await page.goto('/ru');await hydrated
  const carousel=page.getByTestId('opportunity-carousel');await carousel.scrollIntoViewIfNeeded()
  return carousel
}

for(const width of [360,1440])test(`opportunities support trackpad gestures and reachable arrows at ${width}px`,async({page,request})=>{
  const{items}=await(await request.get('/api/creata/v1/public/opportunities?locale=ru')).json();test.skip(items.length<5,'Needs at least five approved opportunities')
  const errors:string[]=[];page.on('pageerror',error=>errors.push(error.message))
  await page.setViewportSize({width,height:950});await page.emulateMedia({reducedMotion:'reduce'})
  const carousel=await load(page),viewport=carousel.locator('[data-carousel-viewport]')
  await expect(carousel).toHaveAttribute('data-current-index','0');await viewport.hover()
  const originalY=await page.evaluate(()=>scrollY)
  await page.mouse.wheel(120,10);await expect(carousel).toHaveAttribute('data-current-index','1')
  expect(await page.evaluate(()=>scrollY)).toBe(originalY)
  // Dispatch the momentum burst together: test-runner round trips must not
  // accidentally turn its tail into a separate gesture after the idle window.
  await page.waitForTimeout(220)
  await viewport.evaluate(element=>{for(const deltaX of [120,40,18])element.dispatchEvent(new WheelEvent('wheel',{deltaX,bubbles:true,cancelable:true}))})
  await expect(carousel).toHaveAttribute('data-current-index','2')
  await page.mouse.wheel(-120,0);await expect(carousel).toHaveAttribute('data-current-index','1')
  const zoomHandled=await viewport.evaluate(element=>!element.dispatchEvent(new WheelEvent('wheel',{deltaX:120,ctrlKey:true,bubbles:true,cancelable:true})))
  expect(zoomHandled).toBe(false);await expect(carousel).toHaveAttribute('data-current-index','1')
  await viewport.dispatchEvent('wheel',{deltaY:100,shiftKey:true,bubbles:true,cancelable:true});await expect(carousel).toHaveAttribute('data-current-index','2')
  const next=carousel.getByRole('button',{name:/Следующий материал/}),previous=carousel.getByRole('button',{name:/Предыдущий материал/})
  await expect(next).toBeVisible();expect((await next.boundingBox())!.y).toBeLessThan((await viewport.boundingBox())!.y)
  await next.click();await expect(carousel).toHaveAttribute('data-current-index','3')
  await previous.click();await expect(carousel).toHaveAttribute('data-current-index','2')
  await viewport.hover();const beforeVertical=await page.evaluate(()=>scrollY)
  await page.mouse.wheel(0,180);await expect.poll(()=>page.evaluate(()=>scrollY)).toBeGreaterThan(beforeVertical)
  await expect(carousel).toHaveAttribute('data-current-index','2')
  expect(await carousel.getByRole('button',{name:/pause|пауза|приостановить/i}).count()).toBe(0)
  expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false);expect(errors).toEqual([])
})

test('site footer contains only Creata branding across language and account routes',async({page})=>{
  for(const route of ['/ru','/kk/profile','/en/about']){
    await page.goto(route);const footer=page.getByRole('contentinfo')
    await expect(footer.getByRole('img',{name:'Creata News',exact:true})).toBeVisible()
    await expect(footer).not.toContainText(/skiper|skipper/i)
    await expect(page.locator('a[href*="skiper-ui"]')).toHaveCount(0)
  }
})
