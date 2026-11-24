/** Public reference fixtures only. Never exports users, secrets or editorial drafts. */
import {readFileSync,writeFileSync,mkdirSync} from 'node:fs'
import {resolve} from 'node:path'
import {copyDemoMedia} from './prepare-demo-assets'
const root=resolve(import.meta.dirname,'..')
const seed=JSON.parse(readFileSync(resolve(root,'.runtime/dev-seed.json'),'utf8'))
const bundle=JSON.parse(readFileSync(resolve(root,'backend/creata/content/pages.json'),'utf8'))
function portable(value:any):any{
 if(Array.isArray(value))return value.map(portable)
 if(value&&typeof value==='object')return Object.fromEntries(Object.entries(value).map(([key,entry])=>[key,portable(entry)]))
 return typeof value==='string'?value.replace(/(?<![\w/])\/(news|newspaper)\/(photos|maps|issues)\//g,'/demo-media/$1/$2/'):value
}
const now=new Date(),date=now.toISOString(),local=(ru:string,kk:string,en:string)=>({ru,kk,en})
const categories=portable(seed.categories),authors=portable(seed.authors)
const opportunities=portable(seed.opportunities.filter((item:any)=>item.status==='approved')).map((item:any,index:number)=>({...item,url:'https://example.org/#creata-demo-'+item.id,deadline:new Date(now.getTime()+(14+index*7)*86400000).toISOString(),languages:['ru','kk','en'],photoRights:{credit:'Creata prototype',rights:'Demo reference'},needsRightsReview:true}))
const articles=portable(seed.articles).map((item:any,index:number)=>({...item,publishedAt:new Date(now.getTime()-index*3600000).toISOString(),createdAt:date,updatedAt:date,author:authors.find((author:any)=>author.id===item.authorId),opportunityIds:index===0?opportunities.slice(0,2).map((opportunity:any)=>opportunity.id):[]}))
const projects=[
 ['atlas-studio','Atlas Studio','creative',local('Рабочее пространство для креативных команд','Креативті командаларға арналған жұмыс кеңістігі','A workspace for creative teams')],
 ['steppe-learn','Steppe Learn','edtech',local('Короткие уроки и первые проекты','Қысқа сабақтар мен алғашқы жобалар','Short lessons and first projects')],
 ['orbit-data','Orbit Data','ai',local('Сложные данные — понятные истории','Күрделі деректер — түсінікті оқиғалар','Complex data, clear stories')],
 ['green-route','Green Route','cities',local('Новый взгляд на городские маршруты','Қалалық бағыттарға жаңа көзқарас','A fresh view of city routes')],
]
const launches=projects.map(([slug,name,categoryId,tagline],index)=>({id:'demo-'+slug,slug,name,categoryId,tagline,description:local('Демонстрационный проект '+name+'. Здесь можно посмотреть продукт, поддержать идею и попробовать обсуждение.',name+' — үлгі жоба. Өнімді қарап, идеяны қолдап, пікір қалдырып көріңіз.',name+' is a fictional demo project. Explore the product, upvote the idea and try the discussion.'),team:local('Демо-команда Creata','Creata демо-командасы','Creata demo team'),url:'https://example.org/',launchDate:date.slice(0,10),country:'KZ',photo:opportunities[index%opportunities.length].photo,status:'published',revision:1,createdAt:date,updatedAt:date,votes:12+index*9,commentsCount:0,tags:['Demo','Creata']}))
articles.push({...articles[0],id:'demo-creative-teams',slug:'demo-creative-teams',format:'graphic',graphicSection:'technology',title:local('Демо: от идеи к первому проекту','Демо: идеядан алғашқы жобаға дейін','Demo: from idea to first project'),lead:local('Пример интерактивного графика на условных данных.','Шартты деректерге негізделген интерактивті график үлгісі.','An interactive chart using fictional sample data.'),body:local('Все значения условные и используются для демонстрации интерфейса.','Барлық мәндер шартты және интерфейсті көрсету үшін берілген.','All values are fictional and used to demonstrate the interface.'),graphic:{type:'bar',labels:[local('Идея','Идея','Idea'),local('Прототип','Прототип','Prototype'),local('Первый запуск','Алғашқы іске қосу','First launch'),local('Развитие','Даму','Growth')],series:[{name:local('Демо-команды','Демо-командалар','Demo teams'),values:[120,85,48,32]}],sourceUrl:'https://example.org/',sourceName:'Creata demo data',unit:'teams',period:'2026',verified:false,methodology:local('Условный набор для проверки графика, не исследование.','Графикті тексеруге арналған шартты деректер.','Fictional data for testing the chart, not research.')}})
const pages=bundle.pages.map((page:any)=>({id:'demo-page-'+page.slug,slug:page.slug,content:portable(page.content),status:'published',revision:1,publishedRevision:1,createdAt:date,updatedAt:date}))
const original=pages.find((page:any)=>page.slug==='newspaper-01').content
original.pdf='/demo-media/newspaper/issues/01/creata-news-01.pdf'
const newspaper={...original,id:'demo-original-01',format:'original',revision:1,publishedRevision:1,entries:[],publishedAt:date}
mkdirSync(resolve(root,'creatanews/src/demo'),{recursive:true})
writeFileSync(resolve(root,'creatanews/src/demo/content.json'),JSON.stringify({categories,authors,articles,opportunities,launches,pages,newspapers:[newspaper]},null,2))
writeFileSync(resolve(root,'creatanews/src/demo/reader-data.json'),JSON.stringify({categories,articles,opportunities,launches},null,2))
copyDemoMedia(resolve(root,'creatanews/public'))
console.log('Demo bundle: '+articles.length+' articles, '+opportunities.length+' opportunities, '+launches.length+' launches, '+pages.length+' site pages; public media copied')
