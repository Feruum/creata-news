import {expect,test,type Locator} from '@playwright/test'
import {openHomeCarousel} from './home-carousel'

async function expectIndex(viewport:Locator,value:number){await expect(viewport).toHaveAttribute('data-current-index',String(value))}

for(const width of [360,1440])for(const kind of ['spotlight','popular'] as const)test(`${kind} carousel supports trackpad, buttons and keyboard at ${width}px`,async({page})=>{
  await page.setViewportSize({width,height:950});await page.emulateMedia({reducedMotion:width===360?'reduce':'no-preference'})
  const {scope,viewport}=await openHomeCarousel(page,kind)
  await viewport.hover()
  await expect(viewport).toHaveAttribute('tabindex','0')
  await expect.poll(async()=>Number(await viewport.getAttribute('data-snap-count'))).toBeGreaterThan(2)
  const y=await page.evaluate(()=>scrollY)
  await page.mouse.wheel(120,10);await expectIndex(viewport,1)
  expect(await page.evaluate(()=>scrollY)).toBe(y)
  await page.waitForTimeout(220)
  await viewport.evaluate(element=>{for(const deltaX of [120,50,20])element.dispatchEvent(new WheelEvent('wheel',{deltaX,bubbles:true,cancelable:true}))})
  await expectIndex(viewport,2)
  await page.mouse.wheel(-120,0);await expectIndex(viewport,1)
  const zoomHandled=await viewport.evaluate(element=>!element.dispatchEvent(new WheelEvent('wheel',{deltaX:120,ctrlKey:true,bubbles:true,cancelable:true})))
  expect(zoomHandled).toBe(false);await expectIndex(viewport,1)
  await viewport.dispatchEvent('wheel',{deltaY:100,shiftKey:true,bubbles:true,cancelable:true});await expectIndex(viewport,2)
  await viewport.focus();await viewport.press('ArrowLeft');await expectIndex(viewport,1)
  await viewport.press('ArrowRight');await expectIndex(viewport,2)
  const previous=scope.getByRole('button',{name:/Предыдущий материал/}),next=scope.getByRole('button',{name:/Следующий материал/})
  await previous.click();await expectIndex(viewport,1);await next.click();await expectIndex(viewport,2)
  await expect(next).toBeEnabled();await expect(previous).toBeEnabled()
  await viewport.focus();await viewport.press('ArrowLeft');await viewport.press('ArrowLeft');await expectIndex(viewport,0)
  await viewport.hover();const before=await page.evaluate(()=>scrollY);await page.mouse.wheel(0,-120)
  await expect.poll(()=>page.evaluate(()=>scrollY)).toBeLessThan(before);await expectIndex(viewport,0)
  expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false)
})

for(const width of [360,1440])test(`related carousel handles trackpad, buttons and keyboard at ${width}px`,async({page,request})=>{
  await page.setViewportSize({width,height:950});await page.emulateMedia({reducedMotion:width===360?'reduce':'no-preference'})
  const {items}=await(await request.get('/api/creata/v1/public/articles?locale=ru')).json()
  const article=items.find((item:any)=>items.some((other:any)=>other.id!==item.id&&other.categoryId===item.categoryId))
  test.skip(!article,'Needs published related stories')
  const errors:string[]=[];page.on('pageerror',error=>errors.push(error.message))
  const hydrated=page.waitForResponse(response=>response.url().endsWith('/auth/me'));await page.goto(`/ru/articles/${article.slug}`);await hydrated
  const carousel=page.locator('.story-carousel'),viewport=carousel.locator('.carousel-viewport')
  // Controlled slide fixture: exercise Embla's live slide observer without
  // adding or modifying articles in the working database.
  await carousel.locator('.carousel-track').evaluate(element=>{
    while(element.children.length<6)element.append(element.firstElementChild!.cloneNode(true))
  })
  await carousel.scrollIntoViewIfNeeded();await viewport.hover()
  await expect(viewport).toHaveAttribute('tabindex','0')
  const previous=carousel.getByRole('button',{name:'Предыдущий материал',exact:true}),next=carousel.getByRole('button',{name:'Следующий материал',exact:true})
  await expect(previous).toBeEnabled();await expect(next).toBeEnabled()
  await page.mouse.wheel(120,0);await expect(carousel).toHaveAttribute('data-current-index','1')
  await page.waitForTimeout(220)
  await viewport.evaluate(element=>{for(const deltaX of [120,50,20])element.dispatchEvent(new WheelEvent('wheel',{deltaX,bubbles:true,cancelable:true}))})
  await expect(carousel).toHaveAttribute('data-current-index','2')
  await page.mouse.wheel(-120,0);await expect(carousel).toHaveAttribute('data-current-index','1')
  await previous.click();await expect(carousel).toHaveAttribute('data-current-index','0');await expect(previous).toBeEnabled()
  await next.click();await expect(carousel).toHaveAttribute('data-current-index','1')
  await viewport.focus();await viewport.press('ArrowRight');await expect(carousel).toHaveAttribute('data-current-index','2')
  await viewport.press('ArrowLeft');await expect(carousel).toHaveAttribute('data-current-index','1')
  await viewport.dispatchEvent('wheel',{deltaY:120,ctrlKey:true,bubbles:true,cancelable:true});await expect(carousel).toHaveAttribute('data-current-index','1')
  for(let i=0;i<6;i++)await viewport.press('ArrowRight')
  await expect(next).toBeEnabled()
  expect(errors).toEqual([])
})
