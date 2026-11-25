import { test, expect } from '@playwright/test'

test.use({ baseURL: process.env.CREATA_FULL_DEMO_TEST_URL, reducedMotion: 'reduce' })
test.skip(!process.env.CREATA_FULL_DEMO_TEST_URL,'Requires the explicit full pipeline demo build')

test('appearance is shared by Editorial and Hermes but not the public website', async ({ page }) => {
  await page.goto('/editorial/settings')
  await expect(page.getByRole('heading',{name:'Внешний вид'})).toBeVisible()
  await expect(page.getByRole('radio',{name:'Светлая',exact:true})).toBeChecked()
  await page.getByRole('radio',{name:'Котики',exact:true}).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme','cats')
  await page.reload()
  await expect(page.getByRole('radio',{name:'Котики',exact:true})).toBeChecked()

  await page.goto('/editorial/today')
  await expect(page.locator('.kanban')).toBeVisible()
  await expect(page.locator('.cats-header-decoration')).toBeVisible()
  const contrast=await page.evaluate(() => {
    const style=getComputedStyle(document.documentElement)
    const luminance=(hex:string)=>{
      const normalized=hex.trim().replace('#','')
      const channels=(normalized.length===3?normalized.split('').map(c=>c+c):normalized.match(/.{2}/g)!).map(c=>parseInt(c,16)/255)
      return channels.reduce((sum,c,i)=>sum+(c<=.04045?c/12.92:((c+.055)/1.055)**2.4)*[.2126,.7152,.0722][i],0)
    }
    const pairs=[['--creata-text','--creata-page'],['--creata-muted','--creata-panel'],['--creata-muted','--creata-surface'],['--creata-accent','--creata-soft'],['--creata-on-accent','--creata-accent']]
    return pairs.map(([a,b])=>{const x=luminance(style.getPropertyValue(a)),y=luminance(style.getPropertyValue(b));return(Math.max(x,y)+.05)/(Math.min(x,y)+.05)})
  })
  for(const value of contrast) expect(value).toBeGreaterThanOrEqual(4.5)

  await page.goto('/hermes/settings')
  await expect(page.getByRole('radio',{name:'Котики',exact:true})).toBeChecked()
  await expect(page.locator('html')).toHaveAttribute('data-theme','cats')
  await page.goto('/hermes/sources')
  await expect(page.getByRole('heading',{name:'Источники',exact:true})).toBeVisible()
  await expect(page.locator('.cats-header-decoration')).toBeVisible()

  await page.goto('/ru')
  await expect(page.locator('html')).not.toHaveAttribute('data-theme','cats')
  await page.goto('/hermes/settings')
  await page.getByRole('radio',{name:'Тёмная',exact:true}).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme','dark')
  await page.goto('/editorial/settings')
  await expect(page.locator('html')).toHaveAttribute('data-theme','dark')
  await page.getByRole('radio',{name:'Светлая',exact:true}).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme','light')
})

test('cats theme keeps the article editor and bank readable',async({page})=>{
 const errors:string[]=[];page.on('pageerror',error=>errors.push(error.message))
 await page.goto('/editorial/settings')
 await page.getByRole('radio',{name:'Котики',exact:true}).click()
 await expect(page.locator('html')).toHaveAttribute('data-theme','cats')
 await page.goto('/editorial/bank')
 await expect(page.getByRole('columnheader',{name:/Оценка Hermes/})).toBeVisible()
 await page.goto('/editorial/articles')
 await page.locator('tbody a.row-title').first().click()
 await expect(page.locator('.editor-main')).toBeVisible()
 await expect(page.locator('.editor-main textarea').first()).toBeVisible()
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:900})
  await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth-innerWidth)).toBeLessThanOrEqual(1)
  expect(await page.locator('.editor-main textarea').first().evaluate(el=>{
   const style=getComputedStyle(el)
   const luminance=(rgb:string)=>rgb.match(/\d+(?:\.\d+)?/g)!.slice(0,3).map(n=>Number(n)/255).reduce((sum,c,i)=>sum+(c<=.04045?c/12.92:((c+.055)/1.055)**2.4)*[.2126,.7152,.0722][i],0)
   const a=luminance(style.color),b=luminance(style.backgroundColor);return(Math.max(a,b)+.05)/(Math.min(a,b)+.05)
  })).toBeGreaterThanOrEqual(4.5)
 }
 expect(errors).toEqual([])
})
