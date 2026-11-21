import {expect,test,type Locator} from '@playwright/test'
import {openHomeCarousel} from './home-carousel'

async function exerciseLoop(viewport:Locator,scope:Locator){
  const previous=scope.getByRole('button',{name:/Предыдущий материал/}),next=scope.getByRole('button',{name:/Следующий материал/})
  await expect(previous).toBeEnabled();await expect(next).toBeEnabled()
  const count=Number(await viewport.getAttribute('data-snap-count'))
  expect(count).toBeGreaterThan(1)
  await expect(viewport).toHaveAttribute('data-current-index','0')
  await previous.click();await expect(viewport).toHaveAttribute('data-current-index',String(count-1))
  await next.click();await expect(viewport).toHaveAttribute('data-current-index','0')
  await viewport.focus()
  for(let step=1;step<=count*2;step++){
    await viewport.press('ArrowRight');await expect(viewport).toHaveAttribute('data-current-index',String(step%count))
  }
  await viewport.hover();await pageWheel(viewport,-120)
  await expect(viewport).toHaveAttribute('data-current-index',String(count-1))
  await pageWheel(viewport,120);await expect(viewport).toHaveAttribute('data-current-index','0')
  await expect(previous).toBeEnabled();await expect(next).toBeEnabled()
}
async function pageWheel(viewport:Locator,deltaX:number){await viewport.page().mouse.wheel(deltaX,0)}

for(const width of [360,1440])for(const kind of ['spotlight','popular'] as const)test(`${kind} loops forward and backward at ${width}px`,async({page})=>{
  await page.setViewportSize({width,height:950});await page.emulateMedia({reducedMotion:width===360?'reduce':'no-preference'})
  const {scope,viewport}=await openHomeCarousel(page,kind)
  await exerciseLoop(viewport,scope)
})

for(const width of [360,1440])test(`related stories loop forward and backward at ${width}px`,async({page,request})=>{
  await page.setViewportSize({width,height:950});await page.emulateMedia({reducedMotion:width===360?'reduce':'no-preference'})
  const {items}=await(await request.get('/api/creata/v1/public/articles?locale=ru')).json()
  const article=items.find((item:any)=>items.some((other:any)=>other.id!==item.id&&other.categoryId===item.categoryId))
  test.skip(!article,'Needs published related stories')
  const hydrated=page.waitForResponse(response=>response.url().endsWith('/auth/me'));await page.goto(`/ru/articles/${article.slug}`);await hydrated
  const carousel=page.locator('.story-carousel')
  await carousel.locator('.carousel-track').evaluate(element=>{while(element.children.length<6)element.append(element.firstElementChild!.cloneNode(true))})
  await carousel.scrollIntoViewIfNeeded()
  await exerciseLoop(carousel.locator('.carousel-viewport'),carousel)
})
