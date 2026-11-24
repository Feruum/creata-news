import {test,expect} from '@playwright/test'
test.use({baseURL:process.env.CREATA_DEMO_TEST_URL,reducedMotion:'reduce'})
test.skip(!process.env.CREATA_DEMO_TEST_URL,'Only run against the explicitly configured frontend demo')

test('public demo pages and media work in three languages without a backend',async({page,request})=>{
 test.setTimeout(180000)
 const errors:string[]=[],failed:string[]=[]
 page.on('pageerror',error=>errors.push(error.message))
 page.on('response',response=>{if(response.status()>=500)failed.push(response.url())})
 for(const locale of ['ru','kk','en'])for(const route of ['','/latest','/opportunities','/launches','/charts','/ecosystem','/about','/gazeta','/authors']){
  await page.goto('/'+locale+route);await expect(page.getByTestId('demo-notice')).toBeVisible()
  await expect(page.getByRole('heading',{level:1})).toHaveCount(1)
  expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false)
 }
 const articles=(await(await request.get('/api/creata/v1/public/articles')).json()).items
 await page.goto('/en/articles/'+articles[0].slug);await expect(page.getByRole('heading',{level:1})).toHaveText(articles[0].title.en)
 await page.goto('/en/search?q='+encodeURIComponent(articles[0].title.en.split(' ')[0]));await expect(page.locator('.story-grid')).toContainText(articles[0].title.en)
 const image=await request.get(articles[0].media.url);expect(image.ok()).toBe(true)
 // The full original issue is loaded in the demo's browser-only content chunk.
 await page.goto('/ru/gazeta/01');await expect(page.locator('.sheet')).toHaveCount(36,{timeout:20000})
 const issue=await(await request.get('/api/creata/v1/public/newspapers/01')).json()
 const pdf=await request.head(issue.pdf);expect(pdf.ok()).toBe(true);expect(pdf.headers()['content-type']).toContain('pdf')
 expect((await request.get('/api/creata/v1/admin/articles')).status()).toBe(404)
 expect((await request.post('/api/creata/v1/account/telegram-link')).status()).toBe(403)
 expect(await(await request.get('/robots.txt')).text()).toContain('Disallow: /')
 expect(errors).toEqual([]);expect(failed).toEqual([])
})

test('demo saves and profile persist locally and stay isolated from another visitor',async({page,request,browser,baseURL})=>{
 const article=(await(await request.get('/api/creata/v1/public/articles')).json()).items[0]
 await page.goto('/en/articles/'+article.slug)
 const save=page.locator('.article-sidebar').getByRole('button',{name:'Save',exact:true});await expect(save).toBeEnabled();await save.click()
 await expect(page.locator('.article-sidebar').getByRole('button',{name:'Remove from saved',exact:true})).toHaveAttribute('aria-pressed','true')
 await page.goto('/en/saved');await expect(page.locator('.story-grid')).toContainText(article.title.en)
 await page.goto('/en/profile');await page.locator('#profile-name').fill('Demo visitor')
 await page.locator('#profile-frequency').selectOption('weekly');await page.getByRole('button',{name:'Save preferences',exact:true}).click()
 await expect(page.getByText('Preferences saved',{exact:true})).toBeVisible();await page.reload()
 await expect(page.locator('#profile-name')).toHaveValue('Demo visitor');await expect(page.locator('#profile-frequency')).toHaveValue('weekly')
 await expect(page.getByText('Telegram connections and digest delivery are disabled in this demo.')).toBeVisible()
 const context=await browser.newContext({baseURL});try{const other=await context.newPage();await other.goto('/en/saved');await expect(other.getByText('Build your own collection',{exact:true})).toBeVisible();await other.goto('/en/profile');await expect(other.locator('#profile-name')).not.toHaveValue('Demo visitor')}finally{await context.close()}
 await page.getByRole('button',{name:'Sign out',exact:true}).click();await expect(page.getByRole('button',{name:'Open demo profile',exact:true})).toBeVisible()
 await page.getByRole('button',{name:'Open demo profile',exact:true}).click();await expect(page.locator('#profile-name')).toHaveValue('Demo visitor')
})

test('mock launch votes and discussion survive refresh only in this browser',async({page,request})=>{
 const launch=(await(await request.get('/api/creata/v1/public/launches')).json()).items[0]
 await page.goto('/en/launches/'+launch.slug)
 const vote=page.getByRole('button',{name:new RegExp('^Upvote: '+launch.name)});await expect(vote).toBeEnabled();await vote.click()
 await expect(page.getByRole('button',{name:new RegExp('^Remove vote: '+launch.name)})).toHaveAttribute('aria-pressed','true')
 await page.getByLabel('Your comment',{exact:true}).fill('Local demo feedback');await page.getByRole('button',{name:'Post comment',exact:true}).click()
 await expect(page.getByText('Local demo feedback',{exact:true})).toBeVisible();await page.reload()
 await expect(page.getByText('Local demo feedback',{exact:true})).toBeVisible()
 await expect(page.getByRole('button',{name:new RegExp('^Remove vote: '+launch.name)})).toContainText(String(launch.votes+1))
 page.on('dialog',dialog=>dialog.accept());await page.getByRole('button',{name:'Delete',exact:true}).click();await expect(page.getByText('Local demo feedback',{exact:true})).toHaveCount(0)
})

test('mobile demo retains Creata layout, themes, localized menu and looping cards',async({page})=>{
 await page.setViewportSize({width:360,height:900});await page.goto('/ru')
 const carousel=page.getByTestId('opportunity-carousel');await carousel.scrollIntoViewIfNeeded();await carousel.locator('[data-carousel-viewport]').hover()
 await page.mouse.wheel(120,0);await expect(carousel).toHaveAttribute('data-current-index','1')
 await page.locator('[aria-controls="creata-menu"]').click();await expect(page.locator('dialog')).toBeVisible();await page.keyboard.press('Escape')
 expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false)
 await page.goto('/kk/latest');await expect(page.getByRole('heading',{level:1})).toHaveText('Соңғы жаңалықтар')
 await expect(page.locator('.bottom-nav')).toBeVisible();await expect(page.locator('a[href*="skiper-ui"]')).toHaveCount(0)
})

test('mobile profile navigation keeps Kazakh labels and arrows inside their links',async({page})=>{
 await page.setViewportSize({width:320,height:780})
 await page.goto('/kk/profile#digest')
 const steps=page.locator('#digest nav a')
 await expect(steps).toHaveCount(3)
 await page.evaluate(()=>document.fonts.ready)

 const overflowing=await steps.evaluateAll(links=>links.filter(link=>{
  const linkRight=link.getBoundingClientRect().right
  return Array.from(link.children).some(child=>child.getBoundingClientRect().right>linkRight)
 }).map(link=>link.textContent))
 expect(overflowing).toEqual([])

 await steps.getByText('Таратылым',{exact:true}).click()
 await expect(page).toHaveURL(/#profile-delivery$/)
 await expect(page.locator('#profile-delivery')).toBeFocused()
})
