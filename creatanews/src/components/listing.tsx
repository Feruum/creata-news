import {getArticleCards,getCategories} from '@/lib/api'
import {publicArticles} from '@/lib/content'
import {ArticleListingView,type ListingProps} from './listing-view'
import {demoMode} from '@/demo/config'
import {DemoListing} from '@/demo/listings'
export async function ArticleListing(props:ListingProps){if(demoMode)return <DemoListing {...props}/>;const{locale,layer,categoryId,region,q,authorId}=props;const[result,categories]=await Promise.all([getArticleCards(locale,{layer,category:categoryId,region,q,authorId}),getCategories()]);return <ArticleListingView {...props} articles={publicArticles(result.data,locale).sort((a,b)=>(b.publishedAt||'').localeCompare(a.publishedAt||''))} categories={categories.data} error={result.error}/>}