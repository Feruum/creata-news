import { expect,test } from 'bun:test'
import { articleTopics,categoryTopic,topics } from './topics'
import type { Article,Category } from './models'

const category=(id:string,ru:string):Category=>({id,slug:id,name:{ru,kk:'',en:''}})
test('imported category aliases share public sections without repeated menu items',()=>{
  expect(categoryTopic(category('cities','Мобильность и умные города'))?.id).toBe('technology')
  expect(categoryTopic(category('mobility','Мобильность'))?.id).toBe('technology')
  expect(categoryTopic(category('stem','Наука и STEM'))?.id).toBe('education')
  expect(categoryTopic(category('science','Наука и STEM'))?.id).toBe('education')
  expect(new Set(topics.map(topic=>topic.id)).size).toBe(topics.length)
})
test('Creata departments do not become public news topics',()=>{
  expect(categoryTopic(category('eco-ventures','Creata Ventures'))).toBeUndefined()
  expect(categoryTopic(category('eco-lab','Creata Lab'))).toBeUndefined()
})
test('article tags can place a story in multiple relevant sections without duplication',()=>{
  const article={categoryId:'ai',tags:['стартап','AI','startups']} as Article
  expect(articleTopics(article,[category('ai','AI и данные')]).map(topic=>topic.id)).toEqual(['startups','technology'])
})
test('clubs and volunteer teams are absent from the current navigation',()=>{
  expect(topics.some(topic=>/клуб|волонт/i.test(topic.name.ru))).toBe(false)
})
