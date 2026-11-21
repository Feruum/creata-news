import {demoMode} from '@/demo/config'
import {DemoAuthorReader} from '@/demo/listings'
import { notFound } from 'next/navigation'
import { ArticleListing } from '@/components/listing'
import { StateMessage } from '@/components/state-message'
import { PublicApiError,serverApi } from '@/lib/api'
import { isLocale,localize } from '@/lib/i18n'
import type { Author } from '@/lib/models'
export default async function Page({params}:{params:Promise<{locale:string;id:string}>}){const{locale:raw,id}=await params;const locale=isLocale(raw)?raw:'ru';if(demoMode)return <DemoAuthorReader locale={locale} id={id}/>;let author:Author;try{author=await serverApi<Author>(`/public/authors/${encodeURIComponent(id)}`)}catch(error){if(error instanceof PublicApiError&&error.status===404)notFound();return <StateMessage locale={locale} error/>}return <ArticleListing locale={locale} title={author.name} description={localize(author.bio,locale)} introduction={<div className='author-avatar'>{author.avatarUrl?<img src={author.avatarUrl} alt=''/>:author.name.charAt(0)}</div>} authorId={id}/>}
