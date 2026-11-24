import {test,expect} from '@playwright/test'
import {resolve} from 'node:path'
import {mkdirSync} from 'node:fs'

test('editor imports financial data, verifies a revision, publishes chart and withdraws it',async({page,context})=>{
 test.setTimeout(120000)
 const origin=process.env.CREATA_TEST_API_ORIGIN||'http://localhost:5555',web=process.env.CREATA_WEB_TEST_URL||'http://localhost:3002',title='Chart QA '+Date.now(),errors:string[]=[]
 page.on('pageerror',e=>errors.push(e.message))
 const login=await context.request.post(`${origin}/api/creata/v1/auth/dev-login`,{data:{role:'admin'}});expect(login.ok()).toBeTruthy()
 const me=await(await context.request.get(`${origin}/api/creata/v1/auth/me`)).json(),headers={'X-CSRF-Token':me.csrfToken}
 expect(me.devAuth).toBe(true)
 const cookie=(await context.cookies()).filter(c=>c.name==='creata_session').map(c=>`${c.name}=${c.value}`).join('; ')
 let id=''
 try{
  await page.goto('/charts');await expect(page.getByRole('heading',{name:'Данные и графики',exact:true})).toBeVisible()
  await page.getByRole('button',{name:'Новый график',exact:true}).click()
  const dialog=page.getByRole('dialog');await dialog.getByLabel('Заголовок',{exact:true}).fill(title)
  await dialog.getByLabel('Рубрика',{exact:true}).selectOption('ai')
  await dialog.getByLabel('Источник',{exact:true}).fill('https://example.org/creata-chart-qa')
  await dialog.getByRole('button',{name:'Сохранить',exact:true}).click()
  await expect(page).toHaveURL(/\/charts\/[a-f0-9]+$/);id=page.url().split('/').pop()!
  await page.getByLabel('Единица измерения',{exact:true}).fill('USD million · QA fixture')
  await page.getByLabel('Период',{exact:true}).fill('2024–2025')
  await page.getByLabel('Название источника',{exact:true}).fill('Controlled development fixture')
  await page.getByText('Импорт CSV',{exact:true}).click()
  await page.getByLabel('CSV',{exact:true}).fill('Period,Revenue\n2024,100\n2025,120')
  await page.getByRole('button',{name:'Применить CSV',exact:true}).click()
  for(const type of ['line','area','donut','bar']){await page.getByLabel('Тип графика',{exact:true}).selectOption(type);await expect(page.locator('.cge-preview .cg-plot svg')).toBeVisible()}
  for(const language of ['RU','ҚАЗ','EN']){
   await page.getByRole('button',{name:language,exact:true}).click()
   await page.getByLabel('Лид',{exact:true}).fill('Development test data; not a real market report.')
   await page.getByLabel('Текст статьи',{exact:true}).fill('Controlled CSV fixture used to verify the editorial chart workflow.')
   await page.getByLabel('Как мы считали',{exact:true}).fill('Two explicitly supplied test values, 100 and 120. No real financial claim.')
  }
  await page.getByLabel('Данные сверены с первоисточником',{exact:true}).check()
  await page.getByRole('button',{name:'Сохранить',exact:true}).click()
  await expect(page.getByText('Редакция 2',{exact:true})).toBeVisible()
  const data=await(await context.request.get(`${origin}/api/creata/v1/admin/articles/${id}`)).json()
  expect(data.graphic.series[0].values).toEqual([100,120]);expect(data.graphic.verified).toBe(true)
  const screenshotDirectory=resolve(import.meta.dirname,'../../.runtime/screenshots')
  mkdirSync(screenshotDirectory,{recursive:true})
  await page.screenshot({path:resolve(screenshotDirectory,'charts-editor-desktop.png'),fullPage:true})
  for(const action of ['Одобрить','Опубликовать']){
   await page.getByRole('button',{name:action,exact:true}).click()
   await page.getByRole('dialog').getByRole('button',{name:action,exact:true}).click()
   await expect(page.getByRole('dialog')).toHaveCount(0)
  }
  const readerContext=await context.browser()!.newContext({viewport:{width:1440,height:1000},reducedMotion:'reduce'}),reader=await readerContext.newPage();reader.on('pageerror',e=>errors.push(e.message))
  await reader.goto(web+'/ru/charts');await expect(reader.locator('[data-chart-ready=true]').first()).toBeVisible();await expect(reader.getByRole('heading',{level:1,name:'Данные и графики',exact:true})).toHaveText('Данные и графики');await expect(reader.getByRole('combobox',{name:'Тип графика'})).toHaveCSS('appearance','none');await expect(reader.getByRole('heading',{name:title,exact:true})).toBeVisible()
  await reader.screenshot({path:resolve(screenshotDirectory,'charts-gallery-desktop.png'),fullPage:true})
  await reader.getByRole('heading',{name:title,exact:true}).getByRole('link').click()
  await expect(reader).toHaveURL(/\/charts\/[^/]+$/,{timeout:30000})
  await expect(reader.getByRole('heading',{name:title,exact:true})).toBeVisible({timeout:30000})
  const chart=reader.locator('.cg-detail > .cg-figure[data-chart-ready=true]')
  await expect(chart).toBeVisible()
  const tableSummary=chart.getByText('Таблица данных',{exact:true})
  const tableValue=chart.getByRole('cell',{name:'120',exact:true})
  await tableSummary.click()
  await expect(tableValue).toBeVisible()
  const download=reader.waitForEvent('download');await chart.getByRole('button',{name:'Скачать CSV',exact:false}).click();expect((await download).suggestedFilename()).toBe('creata-chart.csv')
  await reader.screenshot({path:resolve(screenshotDirectory,'charts-detail-desktop.png'),fullPage:true})
  for(const width of [360,768,1440]){
   await reader.setViewportSize({width,height:width===360?800:900})
   await reader.reload()
   await tableSummary.click()
   await expect(tableValue).toBeVisible()
   const pageWidth=await reader.evaluate(()=>document.documentElement.scrollWidth)
   expect(pageWidth,`Page width with the data table expanded at ${width}px`).toBeLessThanOrEqual(width+1)
   if(width===360)await reader.screenshot({path:resolve(screenshotDirectory,'charts-detail-mobile.png'),fullPage:true})
  }
  await readerContext.close();expect(errors).toEqual([])
 }finally{
  if(id){const root=`${origin}/api/creata/v1/admin/articles/${id}`,requestHeaders={...headers,Cookie:cookie,'Content-Type':'application/json'},current=await(await fetch(root,{headers:requestHeaders,signal:AbortSignal.timeout(5000)})).json();for(const action of [...(current.publishedRevision?['unpublish']:[]),'reject'])await fetch(root+'/actions',{method:'POST',headers:requestHeaders,body:JSON.stringify({action,expectedRevision:current.revision,note:'Development UI verification complete'}),signal:AbortSignal.timeout(5000)})}
 }
})
