import type {schemas} from './generated'
export type {schemas,endpoints} from './generated'
export { articleBodyBlocks, articlePhotoLayout, safeArticlePhotoUrl, safeExternalUrl } from './article-content'
export type Locale = schemas['Locale']
export type LocalizedText = schemas['LocalizedText']
export type SourceRef = schemas['SourceRef']
export type Media = schemas['Media']
export type Author = schemas['Author']
export type Category = schemas['Category']
export type Article = schemas['Article']
export type ArticlePhoto = schemas['ArticlePhoto']
export type ActionGuide = schemas['ActionGuide']
export type StoryUpdate = schemas['StoryUpdate']
export type FollowedStories = schemas['FollowedStories']
export type StoryReadInput = schemas['StoryReadInput']
export type StoryFollowUp = schemas['StoryFollowUp']
export type StoryFollowUpInput = schemas['StoryFollowUpInput']
export type GraphicData = schemas['GraphicData']
export type Opportunity = schemas['Opportunity']
export type Launch = schemas['Launch']
export type LaunchSubmission = schemas['LaunchSubmission']
export type LaunchSubmissionCreate = schemas['LaunchSubmissionCreate']
export type LaunchSubmissionReview = schemas['LaunchSubmissionReview']
export type LaunchComment = schemas['LaunchComment']
export type LaunchVoteState = schemas['LaunchVoteState']
export type NewspaperIssue = schemas['NewspaperIssue']
export type NewspaperItem = schemas['NewspaperItem']
export type NewspaperPublication = schemas['NewspaperPublication']
export type NewspaperCreate = schemas['NewspaperCreate']
export type SitePage = schemas['SitePage']
export type AboutContent = schemas['AboutContent']
export type EcosystemContent = schemas['EcosystemContent']
export type OriginalIssueContent = schemas['OriginalIssueContent']
export type MediaAsset = schemas['MediaAsset']
export type ResearchBrief = schemas['ResearchBrief']
export type StorySimilarity = schemas['StorySimilarity']
export type User = schemas['User']
export type Story = schemas['Story']
export type Source = schemas['Source']
export type Job = schemas['Job']
export type AISettings = schemas['AISettings']
export type Settings = schemas['Settings']
export type Digest = schemas['Digest']
export type MorningDigestStatus = schemas['MorningDigestStatus']
export type Delivery = schemas['Delivery']
export type AuthState = schemas['AuthState']
export type ArticleStatus = Article['status']
export interface List<T> { items: T[]; total?: number }
export class ApiError extends Error { constructor(public status: number, message: string, public code?: string) { super(message); this.name='ApiError' } }
export function createApi(base='/api/creata/v1', csrf?:()=>string|undefined) {
  return async function request<T>(path: string, init: RequestInit={}):Promise<T> {
    const headers=new Headers(init.headers)
    if(init.body && !(init.body instanceof FormData)) headers.set('Content-Type','application/json')
    const token=csrf?.(); if(token) headers.set('X-CSRF-Token',token)
    const res=await fetch(base+path,{...init,headers,credentials:'include'})
    const data=await res.json().catch(()=>({error:res.statusText}))
    if(!res.ok) throw new ApiError(res.status,data.error||res.statusText,data.code)
    return data as T
  }
}
