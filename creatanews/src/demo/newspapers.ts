import type {DemoState} from '@creata/demo-pipeline'
import type {NewspaperIssue} from '@/features/newspaper/catalog'
export function demoNewspaperIssues(state:DemoState):NewspaperIssue[]{
 const all=Object.values(state.newspaperPublications).filter(issue=>issue.slug!=='01')
 const original=state.pagePublications['newspaper-01'];if(original)all.push({...original.content as typeof all[number],id:original.id,revision:original.revision,publishedRevision:original.publishedRevision||1,entries:[],format:'original'})
 return all.map(issue=>({id:issue.slug,number:issue.number,date:issue.date,pages:issue.pages,language:issue.locale,cover:issue.cover.url,headline:issue.description||issue.title,pdf:issue.pdf,title:issue.title})).sort((a,b)=>b.date.localeCompare(a.date)||b.number.localeCompare(a.number))
}
