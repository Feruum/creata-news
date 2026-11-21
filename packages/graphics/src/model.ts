import type {GraphicData,LocalizedText,Locale} from '@creata/contracts'
export type {GraphicData,LocalizedText,Locale}
export const text=(value:LocalizedText|undefined,locale:Locale)=>value?.[locale]||value?.ru||value?.en||value?.kk||''
export const localized=(value=''):LocalizedText=>({ru:value,kk:value,en:value})
export const graphicCopy={
 ru:{title:'Данные и графики',intro:'Рынки, инвестиции и большие изменения — в цифрах.',source:'Источник',period:'Период',methodology:'Как мы считали',table:'Таблица данных',download:'Скачать CSV',inspect:'Для просмотра значений используйте стрелки влево и вправо.',bar:'Столбцы',line:'Линии',area:'Область',donut:'Кольцо',empty:'Графики появятся после редакторской публикации.',all:'Все',finance:'Финансы',technology:'Технологии',world:'Мир',back:'Все графики',unit:'Единица измерения'},
 kk:{title:'Деректер мен графиктер',intro:'Нарықтар, инвестициялар және үлкен өзгерістер — сандармен.',source:'Дереккөз',period:'Кезең',methodology:'Есептеу әдісі',table:'Деректер кестесі',download:'CSV жүктеу',inspect:'Мәндерді көру үшін сол және оң жақ көрсеткілерді пайдаланыңыз.',bar:'Бағандар',line:'Сызықтар',area:'Аймақ',donut:'Сақина',empty:'Графиктер редактор жариялағаннан кейін пайда болады.',all:'Барлығы',finance:'Қаржы',technology:'Технологиялар',world:'Әлем',back:'Барлық графиктер',unit:'Өлшем бірлігі'},
 en:{title:'Data & charts',intro:'Markets, investment and big changes, explained through data.',source:'Source',period:'Period',methodology:'Methodology',table:'Data table',download:'Download CSV',inspect:'Use the left and right arrow keys to inspect values.',bar:'Bars',line:'Lines',area:'Area',donut:'Donut',empty:'Charts will appear after editorial publication.',all:'All',finance:'Finance',technology:'Technology',world:'World',back:'All charts',unit:'Unit'}
} as const
export function validateDataset(g:GraphicData):string|null{
 if(!['bar','line','area','donut'].includes(g.type))return 'Choose a chart type.'
 if(!Array.isArray(g.labels)||g.labels.length<2||g.labels.length>120||!Array.isArray(g.series)||g.series.length<1||g.series.length>6)return 'Use 2–120 rows and 1–6 series.'
 if(g.labels.some(l=>!l||(['ru','kk','en'] as const).some(k=>typeof l[k]!=='string'||!l[k].trim())))return 'Every row label needs RU, ҚАЗ and EN.'
 for(const s of g.series){if(!s.name||(['ru','kk','en'] as const).some(l=>!s.name[l]?.trim()))return 'Every series needs a translated name.';if(s.values.length!==g.labels.length||s.values.some(n=>typeof n!=='number'||!Number.isFinite(n)||Math.abs(n)>1e15))return 'Every row must have a finite numeric value for each series.'}
 if(g.type==='donut'&&(g.series.length!==1||g.series[0].values.some(n=>n<0)||g.series[0].values.reduce((a,b)=>a+b,0)<=0))return 'A donut needs one non-negative series with a positive total.'
 try{const u=new URL(g.sourceUrl);if(!['http:','https:'].includes(u.protocol)||u.username||u.password)throw Error()}catch{return 'Add the original data source URL.'}
 return null
}

// RFC 4180-style quoting, including quoted delimiters and line breaks. Never coerce blank/missing values to zero.
export function parseCsv(input:string):string[][]{
 if(input.length>500_000)throw Error('CSV exceeds 500 KB.')
 const first=input.replace(/^\uFEFF/,'').split(/\r?\n/,1)[0],separator=(first.match(/;/g)||[]).length>(first.match(/,/g)||[]).length?';':','
 const rows:string[][]=[],row:string[]=[];let cell='',quoted=false,closed=false
 const source=input.replace(/^\uFEFF/,'')
 for(let i=0;i<source.length;i++){const c=source[i];if(quoted){if(c==='"'){if(source[i+1]==='"'){cell+='"';i++}else{quoted=false;closed=true}}else cell+=c;continue}
  if(c==='"'){if(cell||closed)throw Error('Invalid CSV quote.');quoted=true;continue}
  if(c===separator||c==='\n'||c==='\r'){row.push(cell.trim());cell='';closed=false;if(c!==separator){if(row.some(Boolean))rows.push([...row]);row.length=0;if(c==='\r'&&source[i+1]==='\n')i++}continue}
  if(closed&&!/\s/.test(c))throw Error('Invalid text after CSV quote.');if(!closed)cell+=c
 }
 if(quoted)throw Error('Unclosed CSV quote.');row.push(cell.trim());if(row.some(Boolean))rows.push(row)
 return rows
}
export function csvToDataset(input:string,existing:GraphicData):GraphicData{
 const rows=parseCsv(input);if(rows.length<3||rows.length>121)throw Error('CSV needs a header and 2–120 data rows.')
 const [header,...values]=rows;if(header.length<2||header.length>7)throw Error('CSV needs a label column and 1–6 numeric columns.')
 if(header.some(h=>!h.trim())||new Set(header).size!==header.length)throw Error('CSV column names must be nonempty and unique.')
 if(values.some(row=>row.length!==header.length||row.some(v=>!v.trim())))throw Error('CSV contains missing cells. Supply every value; zero must be explicit.')
 const numbers=values.map(row=>row.slice(1).map(value=>{if(!/^[+-]?(?:\d+\.?\d*|\.\d+)(?:e[+-]?\d+)?$/i.test(value))throw Error('Numeric CSV cells must use a decimal point and no currency symbols.');const n=Number(value);if(!Number.isFinite(n)||Math.abs(n)>1e15)throw Error('CSV numeric value exceeds limits.');return n}))
 const result={...existing,verified:false,labels:values.map(row=>localized(row[0])),series:header.slice(1).map((name,i)=>({name:localized(name),values:numbers.map(row=>row[i])}))}
 const problem=validateDataset(result);if(problem)throw Error(problem);return result
}
function csvCell(value:string|number){let s=String(value);if(typeof value==='string'&&/^[=+@\-\t\r]/.test(s))s="'"+s;return '"'+s.replaceAll('"','""')+'"'}
export function datasetCsv(g:GraphicData,locale:Locale){return [[graphicCopy[locale].title,...g.series.map(s=>text(s.name,locale))],...g.labels.map((l,i)=>[text(l,locale),...g.series.map(s=>s.values[i])])].map(row=>row.map(csvCell).join(',')).join('\r\n')}
export function numberLabel(value:number,locale:Locale){const [whole,decimal]=value.toFixed(Math.abs(value)<10&&value%1!==0?2:value%1!==0?1:0).split('.');const grouped=whole.replace(/\B(?=(\d{3})+(?!\d))/g,locale==='en'?',':'\u202f');return grouped+(decimal?(locale==='en'?'.':',')+decimal:'')}
export function axisLabel(value:number,locale:Locale){const a=Math.abs(value);const suffix=locale==='en'?['T','B','M','k']:locale==='kk'?['трлн','млрд','млн','мың']:['трлн','млрд','млн','тыс.'];for(const [i,n] of [1e12,1e9,1e6,1e3].entries())if(a>=n)return numberLabel(value/n,locale)+' '+suffix[i];return numberLabel(value,locale)}
export function donutSegments(values:number[]){const circumference=2*Math.PI*101.5,total=values.reduce((a,b)=>a+b,0);let used=0;return values.map(value=>{const length=total>0?value/total*circumference:0,offset=-used;used+=length;return {length:Number(length.toFixed(4)),offset:Number(offset.toFixed(4)),circumference:Number(circumference.toFixed(4))}})}
