import { expect,test } from '@playwright/test'
async function load(page:any){const hydrated=page.waitForResponse((response:any)=>response.url().endsWith('/auth/me'));await page.goto('/ru');await hydrated;const carousel=page.getByTestId('opportunity-carousel');await carousel.scrollIntoViewIfNeeded();return carousel}
test('opportunities autoplay, loop, and pause for hover, focus, hidden document and reduced motion',async({page,request})=>{
  const{items}=await(await request.get('/api/creata/v1/public/opportunities?locale=ru')).json();test.skip(items.length<5,'Needs at least five approved opportunities')
  await page.clock.install();const carousel=await load(page);await page.mouse.move(5,5)
  await expect(carousel).toHaveAttribute('data-autoplay','playing')
  await page.clock.runFor(6500);await expect(carousel).not.toHaveAttribute('data-current-index','0')
  await carousel.hover();await expect(carousel).toHaveAttribute('data-autoplay','paused')
  const hovered=await carousel.getAttribute('data-current-index');await page.clock.fastForward(20000);await expect(carousel).toHaveAttribute('data-current-index',hovered!)
  await page.mouse.move(5,5);const viewport=carousel.locator('[data-carousel-viewport]');await viewport.focus();await expect(carousel).toHaveAttribute('data-autoplay','paused')
  const current=Number(await carousel.getAttribute('data-current-index'))
  for(let i=1;i<=items.length;i++){await viewport.press('ArrowRight');await expect(carousel).toHaveAttribute('data-current-index',String((current+i)%items.length))}
  await page.locator('.brand-link').focus()
  await page.evaluate(()=>{Object.defineProperty(document,'visibilityState',{value:'hidden',configurable:true});document.dispatchEvent(new Event('visibilitychange'))})
  await expect(carousel).toHaveAttribute('data-autoplay','paused');const hidden=await carousel.getAttribute('data-current-index');await page.clock.fastForward(20000);await expect(carousel).toHaveAttribute('data-current-index',hidden!)
  await page.evaluate(()=>{Object.defineProperty(document,'visibilityState',{value:'visible',configurable:true});document.dispatchEvent(new Event('visibilitychange'))})
  await page.emulateMedia({reducedMotion:'reduce'});await expect(carousel).toHaveAttribute('data-autoplay','paused')
  const reduced=await carousel.getAttribute('data-current-index');await page.clock.fastForward(20000);await expect(carousel).toHaveAttribute('data-current-index',reduced!)
})
test('approved opportunities refresh from the API and newly added records join the carousel',async({page,request})=>{
  const{items}=await(await request.get('/api/creata/v1/public/opportunities?locale=ru')).json();test.skip(!items.length,'Needs development opportunities')
  await page.emulateMedia({reducedMotion:'reduce'});await page.clock.install()
  const added={...items[0],id:'controlled-api-addition',title:{...items[0].title,ru:'Новая одобренная возможность из API'}}
  let calls=0;await page.route('**/api/creata/v1/public/opportunities?locale=ru',async route=>{calls++;await route.fulfill({json:{items:[...items,added]}})})
  const carousel=await load(page);await expect(carousel.locator('[data-opportunity-id]')).toHaveCount(items.length)
  await page.clock.fastForward(60001);await expect(carousel.locator('[data-opportunity-id="controlled-api-addition"]')).toContainText(added.title.ru)
  expect(calls).toBeGreaterThan(0);const firstCalls=calls;await page.evaluate(()=>window.dispatchEvent(new Event('focus')));await expect.poll(()=>calls).toBeGreaterThan(firstCalls)
})
