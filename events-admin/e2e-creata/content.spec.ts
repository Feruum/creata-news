import {test,expect} from '@playwright/test'
import {resolve} from 'node:path'

test('site content editor fields stay inside the phone viewport',async({page})=>{
 await page.setViewportSize({width:320,height:800})
 await page.goto('/content')
 await page.getByLabel('Роль',{exact:true}).selectOption('chief_editor')
 await page.getByRole('button',{name:'Войти в локальной разработке',exact:true}).click()
 for(const name of ['Экосистема Creata','О нас','Газета №01']){
  await page.getByRole('button',{name,exact:true}).click()
  const content=page.locator('#main-content')
  await expect(content.getByRole('heading',{name:new RegExp(name)})).toBeVisible()
  await expect(content.locator('fieldset input, fieldset textarea').first()).toBeVisible()
  await expect.poll(()=>content.evaluate(node=>node.scrollWidth-node.clientWidth),{message:name}).toBeLessThanOrEqual(1)
 }
})

test('site text is saved on backend, previewed privately and published without rebuilding',async({page,context})=>{
 test.setTimeout(120000)
 const api=process.env.CREATA_TEST_API_ORIGIN||'http://localhost:5555',web=process.env.CREATA_WEB_TEST_URL||'http://localhost:3002'
 await page.goto('/content?page=about')
 await page.getByLabel('Роль',{exact:true}).selectOption('admin')
 await page.getByRole('button',{name:'Войти в локальной разработке',exact:true}).click()
 await expect(page.getByRole('heading',{name:'Контент сайта',exact:true})).toBeVisible()
 const current=await(await context.request.get(api+'/api/creata/v1/admin/pages/about')).json()
 const marker='Backend editorial page '+Date.now()
 try{
  await page.getByLabel('Заголовок',{exact:true}).first().fill(marker)
  let release!:()=>void,arrived!:()=>void
  const gate=new Promise<void>(resolve=>{release=resolve}),started=new Promise<void>(resolve=>{arrived=resolve})
  await page.route('**/api/creata/v1/admin/pages/about',async route=>{if(route.request().method()==='PATCH'){arrived();await gate}await route.continue()})
  await page.getByRole('button',{name:'Сохранить',exact:true}).click()
  await started
  await expect(page.getByRole('button',{name:'Экосистема Creata',exact:true})).toBeDisabled()
  await expect(page.getByLabel('Заголовок',{exact:true}).first()).toBeDisabled()
  await expect(page.getByRole('button',{name:'EN',exact:true})).toBeDisabled()
  release()
  await expect(page.locator('.section-head h2')).toContainText(String(current.revision+1))
  const publicDraft=await(await context.request.get(api+'/api/creata/v1/public/pages/about')).json()
  expect(publicDraft.content.ru.title).toBe(current.content.ru.title)
  const preview=await context.request.get(web+'/ru/about?preview=1');expect(await preview.text()).toContain(marker)
  await page.getByRole('button',{name:'Опубликовать',exact:true}).click()
  await expect(page.locator('.section-head .badge')).toHaveText('Опубликовано')
  expect(await(await context.request.get(web+'/ru/about')).text()).toContain(marker)
  await page.screenshot({path:resolve(import.meta.dirname,'../../.runtime/screenshots/site-content-editor.png'),fullPage:true})
 }finally{
  const session=await(await context.request.get(api+'/api/creata/v1/auth/me')).json(),headers={'X-CSRF-Token':session.csrfToken}
  const latest=await(await context.request.get(api+'/api/creata/v1/admin/pages/about')).json()
  const restore=await context.request.patch(api+'/api/creata/v1/admin/pages/about',{headers,data:{expectedRevision:latest.revision,content:current.content}});expect(restore.ok()).toBeTruthy()
  const restored=await restore.json();expect((await context.request.post(api+'/api/creata/v1/admin/pages/about/publish',{headers,data:{expectedRevision:restored.revision}})).ok()).toBeTruthy()
 }
})

test('media library uses backend assets and preserves Creata shell in both themes',async({page})=>{
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message))
 await page.goto('/media');await page.getByLabel('Роль',{exact:true}).selectOption('admin');await page.getByRole('button',{name:'Войти в локальной разработке',exact:true}).click()
 await expect(page.getByRole('heading',{name:'Медиатека',exact:true})).toBeVisible()
 await expect(page.locator('tbody tr').first()).toBeVisible()
 await page.locator('.toolbar').getByRole('textbox').fill('cover.png')
 await expect(page.locator('tbody tr')).toHaveCount(1)
 await expect(page.locator('tbody img')).toHaveAttribute('src',/\/media\/imports\//)
 for(const width of [1440,768,360]){await page.setViewportSize({width,height:900});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true)}
 await page.getByRole('button',{name:'Сменить тему',exact:true}).click()
 await expect(page.locator('html')).toHaveAttribute('data-theme','dark')
 await page.getByRole('button',{name:'Сменить тему',exact:true}).click()
 await expect(page.locator('html')).toHaveAttribute('data-theme','light')
 expect(errors).toEqual([])
})
