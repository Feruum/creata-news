import { localize } from './i18n'
import type { Author,Category,Locale,Opportunity } from './models'
export function matchSearchGroups(groups:{authors:Author[];categories:Category[];opportunities:Opportunity[]},query:string,locale:Locale){
  const q=query.trim().toLocaleLowerCase()
  return {
    authors:q?groups.authors.filter(author=>`${author.name} ${localize(author.bio,locale)}`.toLocaleLowerCase().includes(q)):[],
    categories:q?groups.categories.filter(category=>`${localize(category.name,locale)} ${category.slug}`.toLocaleLowerCase().includes(q)):[],
    opportunities:q?groups.opportunities.filter(item=>item.status==='approved'&&`${localize(item.title,locale)} ${localize(item.description,locale)} ${item.organization}`.toLocaleLowerCase().includes(q)):[]
  }
}
export function searchAvailability(errors:boolean[],resultCount:number){const failed=errors.every(Boolean),partial=!failed&&errors.some(Boolean);return {partial,failed,empty:!partial&&!failed&&resultCount===0}}
