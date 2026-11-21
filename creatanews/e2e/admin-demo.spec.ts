import {test,expect} from '@playwright/test'
test.use({baseURL:process.env.CREATA_FULL_DEMO_TEST_URL,reducedMotion:'reduce'})
test.skip(!process.env.CREATA_FULL_DEMO_TEST_URL,'Requires the explicit full pipeline demo build')

test('editorial and Hermes mount existing Creata applications and share navigation',async({page})=>{
 const errors:string[]=[];page.on('pageerror',error=>errors.push(error.message))
 await page.goto('/editorial/today')
 await expect(page.locator('.creata-admin-shell')).toBeVisible()
 await expect(page.getByTestId('admin-demo-notice')).toBeVisible()
 await page.getByTestId('admin-demo-notice').getByRole('link',{name:'Пульт Hermes',exact:true}).click()
 await expect(page).toHaveURL(/\/hermes\/sources$/)
 await expect(page.getByRole('heading',{name:'Источники',exact:true})).toBeVisible()
 await page.getByTestId('admin-demo-notice').getByRole('link',{name:'Редакция',exact:true}).click()
 await expect(page).toHaveURL(/\/editorial\/today$/)
 await page.reload();await expect(page.locator('.kanban')).toBeVisible()
 expect(errors).toEqual([])
})

test('all existing editorial and Hermes sections render without API errors',async({page})=>{
 test.setTimeout(150000)
 const errors:string[]=[];page.on('pageerror',error=>errors.push(error.message))
 for(const [base,sections]of [['editorial',['today','bank','articles','charts','launches','newspapers','opportunities','digests','authors','analytics','content','media','settings','audit']],['hermes',['sources','runs','jobs','charts','rules','ai','errors']]]as const){
  for(const section of sections){
   await page.goto('/'+base+'/'+section)
   await expect(page.locator('.creata-admin-shell')).toBeVisible()
   await expect(page.getByRole('heading',{level:1}).first()).toBeVisible()
   await expect(page.locator('.loading')).toHaveCount(0)
   await expect(page.locator('.error-panel')).toHaveCount(0)
  }
 }
 expect(errors).toEqual([])
})

test('a visitor completes source to Hermes draft to approved publication and simulated digest',async({page,context,browser,baseURL})=>{
 test.setTimeout(150000)
 const marker='Pipeline demo '+Date.now(),errors:string[]=[],external:string[]=[]
 page.on('pageerror',error=>errors.push(error.message))
 page.on('request',request=>{if(request.url().startsWith('https://example.org/'))external.push(request.url())})
 await page.goto('/hermes/sources')
 await page.getByRole('button',{name:'Добавить источник',exact:true}).click()
 const modal=page.getByRole('dialog')
 await modal.getByLabel('Имя',{exact:true}).fill(marker)
 await modal.getByLabel('Ссылка',{exact:true}).fill('https://example.org/rss/'+Date.now())
 await modal.getByRole('button',{name:'Сохранить',exact:true}).click()
 const source=page.locator('.source-card').filter({has:page.getByRole('heading',{name:marker,exact:true})})
 await expect(source).toBeVisible();await source.getByRole('button',{name:'Собрать сейчас',exact:true}).click()
 await page.goto('/hermes/runs')
 const run=page.locator('tbody tr').filter({hasText:marker}).first()
 await expect(run).toBeVisible();await expect(run.locator('.badge')).toHaveText('Завершено',{timeout:25000})
 await page.goto('/editorial/bank')
 const row=page.locator('tbody tr').filter({has:page.getByRole('button',{name:/демо-сюжет/})}).first()
 await expect(row).toBeVisible();await row.getByRole('button',{name:'Взять в работу',exact:true}).click()
 await expect(page).toHaveURL(/\/editorial\/articles\/[^/]+$/)
 await expect(page.getByRole('button',{name:'Одобрить',exact:true})).toBeVisible({timeout:25000})
 const editor=page.locator('.editor-main');await editor.getByRole('button',{name:'RU',exact:true}).click()
 await editor.getByLabel('Заголовок',{exact:true}).fill(marker)
 const slug=await page.getByLabel('Адрес статьи',{exact:true}).inputValue()
 await page.getByRole('button',{name:'Сохранить',exact:true}).click();await expect(page.getByRole('button',{name:'Сохранить',exact:true})).toBeDisabled()
 const publicPage=await context.newPage();publicPage.on('pageerror',error=>errors.push(error.message));await publicPage.goto('/ru/articles/'+slug)
 await expect(publicPage.getByRole('heading',{name:marker,exact:true,level:1})).toHaveCount(0)
 await page.getByRole('button',{name:'На проверку',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'На проверку',exact:true}).click()
 await page.getByRole('button',{name:'Одобрить',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'Одобрить',exact:true}).click()
 await page.getByRole('button',{name:'Опубликовать',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'Опубликовать',exact:true}).click()
 await expect(page.locator('.page-actions .badge')).toHaveText('Опубликовано')
 await publicPage.reload();await expect(publicPage.getByRole('heading',{name:marker,exact:true,level:1})).toBeVisible()
 await publicPage.goto('/ru');await expect(publicPage.locator('[data-testid="home-story-rail"]')).toContainText(marker)
 await publicPage.goto('/ru/latest');await expect(publicPage.locator('#topic-recent')).toHaveCount(1,{timeout:15000});await expect(publicPage.locator('#topic-recent').locator('..')).toContainText(marker)
 const category=await page.getByLabel('Рубрика',{exact:true}).inputValue(),author=await page.getByLabel('Автор',{exact:true}).inputValue(),articlePath=new URL(page.url()).pathname
 await publicPage.goto('/ru/categories/'+category);await expect(publicPage.getByRole('link',{name:marker,exact:true}).first()).toBeVisible()
 await publicPage.goto('/ru/authors/'+author);await expect(publicPage.getByRole('link',{name:marker,exact:true}).first()).toBeVisible()
 await editor.getByLabel('Заголовок',{exact:true}).fill(marker+' unpublished edit')
 await page.getByRole('button',{name:'Сохранить',exact:true}).click();await expect(page.getByRole('button',{name:'Сохранить',exact:true})).toBeDisabled()
 await publicPage.goto('/ru/articles/'+slug);await expect(publicPage.getByRole('heading',{name:marker,exact:true,level:1})).toBeVisible()
 await page.goto('/editorial/digests');await page.getByRole('button',{name:'Собрать выпуск',exact:true}).click()
 await expect(page.locator('.digest-preview')).toContainText(marker)
 await page.getByRole('button',{name:'Одобрить',exact:true}).click();await page.getByRole('button',{name:'Отправить в канал',exact:true}).click()
 await page.getByRole('dialog').getByRole('button',{name:'Отправить в канал',exact:true}).click()
 await expect(page.getByText('Отправлен',{exact:true}).first()).toBeVisible()
 await page.goto(articlePath);await page.getByRole('button',{name:'Снять с публикации',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'Снять с публикации',exact:true}).click()
 for(const path of ['/ru','/ru/latest','/ru/categories/'+category,'/ru/authors/'+author]){await publicPage.goto(path);await expect(publicPage.getByRole('link',{name:marker,exact:true})).toHaveCount(0)}
 const isolated=await browser.newContext({baseURL});try{const other=await isolated.newPage();await other.goto('/hermes/sources');await expect(other.locator('.source-card').filter({hasText:marker})).toHaveCount(0);await other.goto('/ru/articles/'+slug);await expect(other.getByRole('heading',{name:marker,exact:true,level:1})).toHaveCount(0)}finally{await isolated.close()}
 expect(external).toEqual([]);expect(errors).toEqual([])
})

test('new approved opportunities and authors open on public detail pages',async({page,context})=>{
 const marker='Opportunity demo '+Date.now()
 await page.goto('/editorial/opportunities');await page.getByRole('button',{name:'Добавить возможность',exact:true}).click()
 const modal=page.getByRole('dialog')
 await modal.getByLabel('Заголовок',{exact:true}).fill(marker);await modal.getByLabel('Описание',{exact:true}).fill('Browser-local opportunity description')
 await modal.getByLabel('Организация',{exact:true}).fill('Creata Demo');await modal.getByLabel('Страна',{exact:true}).fill('KZ');await modal.getByLabel('Дедлайн',{exact:true}).fill('2027-12-31T09:00');await modal.getByLabel('Ссылка',{exact:true}).fill('https://example.org/demo-opportunity');await modal.getByLabel('Статус',{exact:true}).selectOption('approved');await modal.getByRole('button',{name:'Сохранить',exact:true}).click()
 await expect(page.getByRole('button',{name:marker,exact:true})).toBeVisible()
 const publicPage=await context.newPage();await publicPage.goto('/ru/opportunities');await publicPage.getByRole('link',{name:marker,exact:true}).first().click();await expect(publicPage.getByRole('heading',{level:1})).toHaveText(marker)
 const author='Author demo '+Date.now();await page.goto('/editorial/authors');await page.getByRole('button',{name:'Добавить автора',exact:true}).click();await page.getByRole('dialog').getByLabel('Имя',{exact:true}).fill(author);await page.getByRole('dialog').getByRole('button',{name:'Сохранить',exact:true}).click()
 await expect(page.getByRole('heading',{name:author,exact:true})).toBeVisible();await publicPage.goto('/ru/authors');await publicPage.getByRole('link').filter({has:publicPage.getByRole('heading',{name:author,exact:true})}).click();await expect(publicPage.getByRole('heading',{level:1})).toHaveText(author)
})

test('existing Creata admin layout works at phone tablet and desktop sizes',async({page})=>{
 for(const width of [360,768,1440]){
  await page.setViewportSize({width,height:900})
  for(const path of ['/editorial/today','/hermes/sources']){
   await page.goto(path);await expect(page.locator('.creata-admin-shell')).toBeVisible();await expect(page.getByRole('heading',{level:1}).first()).toBeVisible();await expect(page.locator('.loading')).toHaveCount(0)
   expect(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)).toBe(false)
  }
 }
})

test('demo CMS keeps unpublished edits in preview and publishes the selected revision',async({page,context})=>{
 const title='CMS demo '+Date.now()
 await page.goto('/editorial/content');await page.getByRole('button',{name:'О нас',exact:true}).click()
 await page.getByLabel('Заголовок',{exact:true}).fill(title)
 await page.getByRole('button',{name:'Сохранить',exact:true}).click()
 await expect(page.getByRole('button',{name:'Сохранить',exact:true})).toBeDisabled()
 const preview=await context.newPage();await preview.goto('/ru/about?preview=1');await expect(preview.getByRole('heading',{level:1})).toHaveText(title)
 await preview.goto('/ru/about');await expect(preview.getByRole('heading',{level:1})).not.toHaveText(title)
 await page.getByRole('button',{name:'Опубликовать',exact:true}).click();await preview.reload()
 await expect(preview.getByRole('heading',{level:1})).toHaveText(title)
})
