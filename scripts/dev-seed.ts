/** Explicit development seed conversion; never invoked by production startup. */
import {readFileSync,mkdirSync,writeFileSync} from 'node:fs'
import {runInNewContext} from 'node:vm'
const context={window:{} as any}
runInNewContext(readFileSync('creata-news-v2/site-v2/assets/data.js','utf8'),context)
const original=context.window.CN_SEED
const names:Record<string,string>={ai:'ЖИ және деректер',startups:'Стартаптар мен венчур',stem:'Ғылым және STEM',robotics:'Робототехника',space:'Ғарыш',climate:'Климат және әсер',health:'Медицина және биотехнология',edtech:'Білім беру',fintech:'Қаржы технологиялары',cities:'Қалалар және көлік',agri:'Ауыл шаруашылығы',creative:'Креативті индустриялар','eco-group':'Creata Group','eco-transfornation':'TransforNATION','eco-lab':'Creata Lab','eco-games':'Creata Games','eco-ciaq':'CIAQ және GCA','eco-ventures':'Creata Ventures','eco-team':'Creata Team'}
const kk:Record<string,[string,string]>={
 'w-ucode':['Ташкенттің Ucode платформасы: әзірлеушілерге арналған ЖИ','Udevs негізін қалаушылар әзірлеушілерге арналған платформаға өз қаражатынан шамамен 300 мың доллар инвестиция салып, АҚШ нарығына шықты.'],
 'w-uz-hk':['Өзбекстан стартаптары Hong Kong FinTech Week көрмесінде','IT Park Гонконгқа Tass Vision, OX System, Unitlab AI, Aziza AI және Qwatt жобаларын таныстырды.'],
 'w-kenya-ai':['Найробидегі ЖИ: қолжетімді техника және ым тілі','Кениядағы технология көрмесінде стартаптар арзан құрылғыларда жұмыс істейтін және инклюзивтілікті арттыратын ЖИ шешімдерін ұсынды.'],
 'w-opin':['Бразилия мен Үндістан OPIN желісін іске қосты','Екі ел климат мәселелеріне арналған ортақ цифрлық инфрақұрылымды ұсынады: ерте ескертуден фермерлерге төлем жасауға дейін.'],
 'w-tokyo-humanoids':['Токиодағы гуманоид роботтар','Жапон әзірлеушілері әуежайларға арналған роботтар мен инеге жіп өткізетін робот қолын көрсетті.'],
 'a-vahta':['Айранкөл мұнайшыларының вахтадағы өмірі','Forbes.kz фоторепортажы Атырау облысындағы кен орнында жұмыс пен тұрмыстың қалай ұйымдастырылғанын көрсетеді.'],
 'a-akku':['Алматыдағы «Аққу» кафесінің болашағы','Жазушылар, суретшілер мен студенттер кездескен тарихи кафе қайта жаңғыртылуда. Қала тұрғындары оның болашағын талқылайды.'],
 'a-mosques':['Алматы мешіттерінің сегіз ғасырлық тарихы','BES.media Алматы мешіттерінің ортағасырлық қалашықтан қазіргі ірі кешендерге дейінгі өзгерісін зерттеді.']
}
const categories=original.verticals.map((v:any)=>({id:v.id,slug:v.id,name:{ru:v.name,en:v.name_en||v.name,kk:names[v.id]||v.name}}))
const authors=original.authors.map((a:any)=>({id:a.id,name:a.name,bio:{ru:a.bio||'',en:a.bio_en||a.bio||'',kk:'Creata News редакциясының авторы.'},role:a.role||'',avatarUrl:''}))
function text(a:any,key:string){const ru=a[key]||'',en=a[key+'_en']||ru;return {ru,en,kk:key==='title'?kk[a.id]?.[0]||a.title:key==='lead'?kk[a.id]?.[1]||a.lead:kk[a.id]?.[1]||''}}
const articles=original.articles.filter((a:any)=>kk[a.id]&&a.status==='published').map((a:any,i:number)=>({id:a.id,slug:a.slug||a.id,title:text(a,'title'),lead:text(a,'lead'),body:{ru:(a.blocks||[]).filter((b:any)=>b.t==='p').map((b:any)=>b.text).join('\n\n')||a.lead,en:(a.blocks_en||[]).filter((b:any)=>b.t==='p').map((b:any)=>b.text).join('\n\n')||a.lead_en||a.lead,kk:kk[a.id][1]},summary:text(a,'lead'),actions:{ru:'Изучите первоисточник и связанные возможности.',kk:'Түпнұсқаны және байланысты мүмкіндіктерді қараңыз.',en:'Read the original source and explore related opportunities.'},layer:a.layer==='eco'?'eco':'world',categoryId:a.vertical,region:a.country||'KZ',tags:a.tags||[],authorId:a.author,sources:a.source?.url?[{url:a.source.url,title:a.source.name,publishedAt:a.source.date?new Date(a.source.date).toISOString():undefined}]:[],reviewNotes:['Development fixture from the supplied reference prototype; not a live editorial publication.'],media:a.photo?{url:a.photo.url.replace('assets/','/news/'),caption:a.photo.credit||'',credit:a.photo.credit||'',rights:'Development reference only; verify reuse rights before production.'}:undefined,status:'published',revision:1,publishedAt:new Date(Date.UTC(2026,10,13,12-i)).toISOString()}))
const opportunityKk:Record<string,{title:string,audience:string,conditions:string,category:string}>={
 'o-designthon':{title:'Designthon: TransforNATION дизайн марафоны',audience:'Студенттер мен жас дизайнерлер',conditions:'Төрт адамға дейінгі командалар. Финал офлайн өтеді.',category:'creative'},
 'o-creative-start':{title:'Creative Start: креативті индустриялардағы алғашқы жоба',audience:'Жоғары сынып оқушылары мен студенттер',conditions:'Онлайн өтінім және индустрия тәлімгерлері.',category:'creative'},
 'o-gamejam':{title:'Creata Games геймджемі: 48 сағатта ойын жасау',audience:'Әзірлеушілер, суретшілер және ойын дизайнерлері',conditions:'Жеке немесе командамен қатысуға болады.',category:'eco-games'},
 'o-academy':{title:'Creata Academy тағылымдамасы',audience:'EdTech және педагогика студенттері',conditions:'Аптасына 8–11 сағат.',category:'edtech'},
 'o-stem-week':{title:'Invent Future Global: STEM жобасына грант',audience:'Оқушылар мен студенттер командалары',conditions:'Прототип және екі минуттық бейне таныстырылым.',category:'stem'},
 'o-mamapro':{title:'«Мама Про» хакатоны',audience:'Кәсіпкер аналар мен олардың командалары',conditions:'Редактордың тексеруін күтуде.',category:'startups'}
}
const opportunityTypes:Record<string,string>={'Конкурс':'contest','Программа':'program','Хакатон':'hackathon','Стажировка':'internship','Грант':'grant'}
// Prototype DEMO records keep their review status; pending hackathons never enter the public feed.
const opportunities=original.opportunities.filter((o:any)=>opportunityKk[o.id]).map((o:any)=>({
 id:o.id,title:{ru:o.title,en:o.title_en,kk:opportunityKk[o.id].title},
 description:{ru:o.conditions,en:o.conditions_en,kk:opportunityKk[o.id].conditions},
 organization:o.org,type:opportunityTypes[o.type]||o.type,url:o.url,country:o.country==='Онлайн'?'Online':o.country==='Весь мир'?'Worldwide':'KZ',
 deadline:new Date(o.deadline).toISOString(),status:o.status,categoryId:opportunityKk[o.id].category,
 photo:o.photo?.replace('assets/','/news/'),forWhom:{ru:o.forWhom,en:o.forWhom_en,kk:opportunityKk[o.id].audience}
}))
const seed={categories,authors,articles,opportunities}
mkdirSync('.runtime',{recursive:true});writeFileSync('.runtime/dev-seed.json',JSON.stringify(seed,null,2));console.log(`Development seed: ${articles.length} articles, ${categories.length} categories, ${authors.length} authors, ${opportunities.length} opportunities`)
