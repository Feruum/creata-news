import {test,expect} from '@playwright/test'

const siteURL=process.env.CREATA_HOME_TEST_URL||process.env.CREATA_WEB_TEST_URL
test.use({baseURL:siteURL})
test.skip(!siteURL,'Requires the local Creata site or an explicit homepage preview')

test('the site menu can reopen while the next page is loading',async({page})=>{
  let finishNavigation=()=>{}
  const pendingPage=new Promise<void>(resolve=>{finishNavigation=resolve})
  await page.route('**/ru/profile?**',async route=>{
    await pendingPage
    await route.continue()
  })

  try{
    await page.goto('/ru/latest?section=technology')
    const opener=page.getByRole('button',{name:'Открыть меню'})
    const menu=page.locator('#creata-menu')
    await opener.click()
    await menu.locator('a[href="/ru/profile"]').click()

    expect(await menu.evaluate((dialog:HTMLDialogElement)=>dialog.open)).toBe(false)
    expect(await page.evaluate(()=>document.body.style.overflow)).toBe('')
    await opener.click()
    await expect(menu).toBeVisible()

    finishNavigation()
    await expect(page).toHaveURL('/ru/profile')
    await expect(page.locator('#profile-heading')).toBeVisible()
    await expect(menu).toBeVisible()

    await menu.getByRole('button',{name:'Закрыть'}).click()
    await expect(menu).toBeHidden()
    await expect(opener).toHaveAttribute('aria-expanded','false')
  }finally{
    finishNavigation()
    await page.unrouteAll({behavior:'ignoreErrors'})
  }
})

test('the profile still contains the digest after returning from the menu',async({page})=>{
  await page.goto('/ru/profile#digest')
  await expect(page.locator('#profile-heading')).toBeVisible()
  const opener=page.getByRole('button',{name:'Открыть меню'})
  const menu=page.locator('#creata-menu')
  await opener.click()
  await menu.getByRole('link',{name:'Технологии',exact:true}).click()
  await expect(page).toHaveURL(/\/ru\/latest\?section=technology$/)
  await opener.click()
  await menu.locator('a[href="/ru/profile"]').click()
  await expect(page).toHaveURL(/\/ru\/profile(?:#digest)?$/)
  await expect(page.locator('#digest #profile-heading')).toHaveText('Утро Creata')
  await opener.click()
  await expect(menu).toBeVisible()
})

test('menu links remain opaque after navigating between the newspaper and startup launches',async({page})=>{
  await page.goto('/ru/charts')
  const opener=page.getByRole('button',{name:'Открыть меню'})
  const menu=page.locator('#creata-menu')
  const regions=menu.getByRole('navigation',{name:'Регионы'})
  await opener.click()
  await expect(regions.getByRole('link').first()).toHaveCSS('opacity','1')

  for(const [name,path] of [['Газета','/ru/gazeta'],['Запуски стартапов','/ru/launches']]){
    await menu.getByRole('link',{name,exact:true}).click()
    await expect(page).toHaveURL(path)
    await opener.click()
    await expect(menu).toHaveCSS('opacity','1')
    await expect(regions.getByRole('link').first()).toHaveCSS('opacity','1')
  }
})

test('the public menu keeps four sections and routes readers to the ecosystem through About',async({page})=>{
  await page.goto('/ru')
  const header=page.locator('.site-header')
  await expect(header.locator('a[href="/ru/v2"], a[href="/ru/ecosystem"], a[href="/ru/profile#digest"]')).toHaveCount(0)
  await header.getByRole('button',{name:'Открыть меню'}).click()
  const menu=page.locator('#creata-menu')
  const sections=menu.getByRole('navigation',{name:'Разделы',exact:true})
  await expect(sections.getByRole('link')).toHaveCount(4)
  for(const [name,path] of [['Главная','/ru'],['Газета','/ru/gazeta'],['Возможности','/ru/opportunities'],['Запуски стартапов','/ru/launches']]){
    await expect(sections.getByRole('link',{name,exact:true})).toHaveAttribute('href',path)
  }
  await expect(menu.locator('a[href="/ru/v2"], a[href="/ru/charts"], a[href="/ru/ecosystem"], a[href="/ru/profile#digest"]')).toHaveCount(0)

  await menu.getByRole('navigation',{name:'Быстрые ссылки'}).getByRole('link',{name:'О Creata',exact:true}).click()
  await expect(page).toHaveURL('/ru/about')
  const ecosystem=page.getByRole('main').getByRole('link',{name:'Экосистема Creata',exact:true})
  await expect(ecosystem).toHaveAttribute('href','/ru/ecosystem')
  await ecosystem.click()
  await expect(page).toHaveURL('/ru/ecosystem')
  await expect(page.getByTestId('ecosystem-page')).toBeVisible()
})

test('the regions menu offers Central Asia and combines legacy country names in its collections',async({page,request})=>{
  const response=await request.get('/api/creata/v1/public/articles?locale=ru')
  await expect(response).toBeOK()
  const {items}=await response.json() as {items:{region:string;title:{ru:string}}[]}
  await page.goto('/ru/gazeta')
  const opener=page.getByRole('button',{name:'Открыть меню'})
  const regions=page.locator('#creata-menu').getByRole('navigation',{name:'Регионы'})
  await opener.click()
  await expect(regions.getByRole('link')).toHaveText(['Центральная Азия','Казахстан','Кыргызстан','Таджикистан','Туркменистан','Узбекистан'])

  await regions.getByRole('link',{name:'Центральная Азия',exact:true}).click()
  await expect(page).toHaveURL('/ru/latest?region=central-asia')
  await expect(page.getByRole('heading',{level:1,name:'Центральная Азия'})).toBeVisible()
  const collection=page.getByRole('region',{name:'Центральная Азия',exact:true})
  for(const item of items.filter(article=>['KZ','Kazakhstan','UZ','Uzbekistan'].includes(article.region)&&article.title.ru)){
    await expect(collection.getByRole('heading',{name:item.title.ru,exact:true})).toBeVisible()
  }
  for(const item of items.filter(article=>['Global','JP','KE','BR'].includes(article.region)&&article.title.ru)){
    await expect(collection.getByRole('heading',{name:item.title.ru,exact:true})).toHaveCount(0)
  }

  await opener.click()
  await regions.getByRole('link',{name:'Казахстан',exact:true}).click()
  await expect(page).toHaveURL('/ru/latest?region=KZ')
  const kazakhstan=page.getByRole('region',{name:'Казахстан',exact:true})
  for(const item of items.filter(article=>['KZ','Kazakhstan','Казахстан'].includes(article.region)&&article.title.ru)){
    await expect(kazakhstan.getByRole('heading',{name:item.title.ru,exact:true})).toBeVisible()
  }
})
