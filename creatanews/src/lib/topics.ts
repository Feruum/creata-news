import type { Article, Category, Locale } from './models'
import { copy } from './i18n'

// Public navigation groups, independent of imported category IDs and Creata departments.
export const topics = [
  {id:'startups',name:{ru:'Стартапы',kk:'Стартаптар',en:'Startups'},aliases:['startup','startups','venture','ventures','fintech'],pattern:/стартап|венчур|\bstartups?\b|\bventure\b|fintech/iu},
  {id:'technology',name:{ru:'Технологии',kk:'Технологиялар',en:'Technology'},aliases:['ai','robotics','cities','mobility','technology'],pattern:/технолог|робот|мобильност|\bai\b|\bdata\b|\brobot|\bcities\b|\bmobility\b|ақпарат/iu},
  {id:'education',name:{ru:'Наука и образование',kk:'Ғылым және білім',en:'Science & education'},aliases:['science','stem','space','education','edtech'],pattern:/наук|образован|космос|ғылым|білім|ғарыш|\bscience\b|\bstem\b|\bspace\b|\beducation\b|\bedtech\b/iu},
  {id:'creative',name:{ru:'Креативные индустрии',kk:'Креативті индустриялар',en:'Creative industries'},aliases:['creative','culture','art'],pattern:/креатив|культур|искусств|creative|culture|\bart\b/iu},
  {id:'impact',name:{ru:'Общество и импакт',kk:'Қоғам және ықпал',en:'Society & impact'},aliases:['climate','health','food','agri','impact'],pattern:/климат|здоров|биотех|агро|climate|health|agri|food|impact|импакт/iu},
] as const

export type Topic = typeof topics[number]
export function findTopic(value?:string):Topic|undefined { return topics.find(topic=>topic.id===value) }
export function categoryTopic(category:Category):Topic|undefined {
  if (/^eco[-_]/i.test(category.id)||/^eco[-_]/i.test(category.slug)) return
  const identifiers=[category.id,category.slug].filter(Boolean).map(value=>value.toLowerCase())
  const exact=topics.find(topic=>identifiers.some(value=>(topic.aliases as readonly string[]).includes(value)))
  return exact||topics.find(topic=>topic.pattern.test(Object.values(category.name).join(' ')))
}
export function articleTopics(article:Article,categories:Category[]):Topic[] {
  const category=categories.find(item=>item.id===article.categoryId)
  const primary=category?categoryTopic(category):topics.find(topic=>(topic.aliases as readonly string[]).includes(article.categoryId))
  const tags=article.tags.join(' ')
  return topics.filter(topic=>topic.id===primary?.id||topic.pattern.test(tags))
}
export function topicLabel(topic:Topic,locale:Locale) { return topic.name[locale] }
const descriptions:Record<Topic['id'],Record<Locale,string>>={
  startups:{ru:'Новые продукты, команды и идеи. Как стартапы находят первых пользователей и превращают задумки в работающий бизнес.',kk:'Жаңа өнімдер, командалар мен идеялар. Стартаптар алғашқы пайдаланушыларын қалай тауып, идеяларын бизнеске айналдырады.',en:'New products, teams and ideas. How startups find their first users and build ideas into working businesses.'},
  technology:{ru:'Искусственный интеллект, робототехника и технологии, которые меняют повседневную жизнь. Разбираемся, как они работают и что открывают для созидателей.',kk:'Жасанды интеллект, робототехника және күнделікті өмірді өзгертетін технологиялар. Олардың қалай жұмыс істейтінін және қандай мүмкіндік ашатынын зерттейміз.',en:'AI, robotics and technologies changing everyday life. How they work and what they make possible for creators.'},
  education:{ru:'Наука, исследования и новые способы учиться. Люди и проекты, которые помогают лучше понимать мир и создавать новое.',kk:'Ғылым, зерттеулер және оқудың жаңа тәсілдері. Әлемді түсінуге және жаңалық жасауға көмектесетін адамдар мен жобалар.',en:'Science, research and new ways to learn. People and projects helping us understand the world and create something new.'},
  creative:{ru:'Искусство, дизайн, культура и креативные индустрии. Истории людей, которые создают новые формы, смыслы и впечатления.',kk:'Өнер, дизайн, мәдениет және креативті индустриялар. Жаңа пішіндер, мағыналар мен әсерлер жасайтын адамдардың оқиғалары.',en:'Art, design, culture and creative industries. Stories of people making new forms, meanings and experiences.'},
  impact:{ru:'Инициативы, которые меняют жизнь людей, городов и сообществ. Климат, здоровье и практические решения общественных задач.',kk:'Адамдардың, қалалар мен қауымдастықтардың өмірін өзгертетін бастамалар. Климат, денсаулық және қоғамдық мәселелердің шешімдері.',en:'Ideas changing life for people, cities and communities. Climate, health and practical responses to shared challenges.'},
}
export function topicDescription(topic:Topic,locale:Locale) { return descriptions[topic.id][locale] }
// Explicit labels keep server and browser HTML identical when Kazakh ICU data
// differs. Unknown values remain the source's code instead of changing locale.
const regions:Record<string,Record<Locale,string>>={
  KZ:{ru:'Казахстан',kk:'Қазақстан',en:'Kazakhstan'},UZ:{ru:'Узбекистан',kk:'Өзбекстан',en:'Uzbekistan'},KG:{ru:'Кыргызстан',kk:'Қырғызстан',en:'Kyrgyzstan'},TJ:{ru:'Таджикистан',kk:'Тәжікстан',en:'Tajikistan'},TM:{ru:'Туркменистан',kk:'Түрікменстан',en:'Turkmenistan'},
  RU:{ru:'Россия',kk:'Ресей',en:'Russia'},CN:{ru:'Китай',kk:'Қытай',en:'China'},JP:{ru:'Япония',kk:'Жапония',en:'Japan'},IN:{ru:'Индия',kk:'Үндістан',en:'India'},KR:{ru:'Южная Корея',kk:'Оңтүстік Корея',en:'South Korea'},HK:{ru:'Гонконг',kk:'Гонконг',en:'Hong Kong'},
  KE:{ru:'Кения',kk:'Кения',en:'Kenya'},NG:{ru:'Нигерия',kk:'Нигерия',en:'Nigeria'},ZA:{ru:'ЮАР',kk:'Оңтүстік Африка',en:'South Africa'},BR:{ru:'Бразилия',kk:'Бразилия',en:'Brazil'},US:{ru:'США',kk:'АҚШ',en:'United States'},CA:{ru:'Канада',kk:'Канада',en:'Canada'},
  GB:{ru:'Великобритания',kk:'Ұлыбритания',en:'United Kingdom'},DE:{ru:'Германия',kk:'Германия',en:'Germany'},FR:{ru:'Франция',kk:'Франция',en:'France'},NL:{ru:'Нидерланды',kk:'Нидерланд',en:'Netherlands'},AE:{ru:'ОАЭ',kk:'БАӘ',en:'UAE'},
  ONLINE:{ru:'Онлайн',kk:'Онлайн',en:'Online'},WORLDWIDE:{ru:'Весь мир',kk:'Бүкіл әлем',en:'Worldwide'},GLOBAL:{ru:'Весь мир',kk:'Бүкіл әлем',en:'Worldwide'}
}

export const centralAsiaRegion = 'central-asia'
export const centralAsianCountries = ['KZ', 'KG', 'TJ', 'TM', 'UZ'] as const
const centralAsiaNames = new Set([centralAsiaRegion, ...Object.values(copy).map(text => text.centralAsia.toLowerCase())])

function centralAsiaCode(value: string) {
  if (typeof value !== 'string') return
  const name = value.trim().toLowerCase()
  if (centralAsiaNames.has(name)) return centralAsiaRegion
  return centralAsianCountries.find(code => code.toLowerCase() === name ||
    Object.values(regions[code]).some(label => label.toLowerCase() === name))
}

export function matchesArticleRegion(articleRegion: string, requestedRegion: string) {
  const requested = centralAsiaCode(requestedRegion)
  if (!requested) return articleRegion === requestedRegion
  const article = centralAsiaCode(articleRegion)
  return requested === centralAsiaRegion ? article !== undefined : article === requested
}

export function regionLabel(value:string,locale:Locale):string {
  const code = centralAsiaCode(value)
  if (code === centralAsiaRegion) return copy[locale].centralAsia
  return regions[code ?? value.toUpperCase()]?.[locale] || value
}
