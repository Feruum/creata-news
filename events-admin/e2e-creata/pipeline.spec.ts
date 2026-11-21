import {test,expect} from '@playwright/test'

test('newsroom can see stage durations, duplicate review and persisted Hermes research',async({page,context})=>{
 const api=process.env.CREATA_TEST_API_ORIGIN||'http://localhost:5555'
 await page.goto('/analytics');await page.getByLabel('Роль',{exact:true}).selectOption('admin');await page.getByRole('button',{name:'Войти в локальной разработке',exact:true}).click()
 await expect(page.getByRole('heading',{name:'Длительность редакционных этапов',exact:true})).toBeVisible()
 await expect(page.getByRole('heading',{name:'Доля материалов, достигших этапа',exact:true})).toBeVisible()
 await page.goto('/bank');await expect(page.getByRole('heading',{name:'Похожие сюжеты',exact:true})).toBeVisible()
 const articles=(await(await context.request.get(api+'/api/creata/v1/admin/articles')).json()).items as any[]
 const article=articles.find(a=>a.researchBrief?.facts?.length)
 expect(article,'Controlled worker fixtures must leave a persisted source-backed draft').toBeTruthy()
 await page.goto('/articles/'+article.id)
 await expect(page.getByRole('heading',{name:'Этапы Hermes',exact:true})).toBeVisible()
 await expect(page.getByRole('heading',{name:'Исследовательская справка',exact:true}).first()).toBeVisible()
 await expect(page.getByText(article.researchBrief.facts[0].claim,{exact:true}).first()).toBeVisible()
})
