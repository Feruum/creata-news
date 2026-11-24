import type { Article,Category,Locale } from '@/lib/models'
import { localize } from '@/lib/i18n'
import { articleReadingMinutes } from '@/lib/content'
import {homeStoryImage} from './home-story-images'
// Shared presentation data for the editorial lead, latest list and story rail.
export function adaptArticleForHome(article:Article,locale:Locale,categories:Category[]){
  return {id:article.id,slug:article.slug,title:localize(article.title,locale),lead:localize(article.lead,locale),imageUrl:homeStoryImage(article),imageCredit:homeStoryImage(article)===article.media?.url?article.media?.credit:undefined,categoryId:article.categoryId,categoryName:localize(categories.find(category=>category.id===article.categoryId)?.name,locale),authorId:article.authorId,authorName:article.author?.name||'',publishedAt:article.publishedAt,readMinutes:articleReadingMinutes(article,locale),sourceCount:article.sources.length,layer:article.layer,region:article.region}
}
export type HomeStory=ReturnType<typeof adaptArticleForHome>
