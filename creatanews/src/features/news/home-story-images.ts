import type {Article} from '@/lib/models'

// Homepage presentation only. Preserve published photos and the original media
// in article pages, collections, search and CMS records.
// Credits and original asset URLs: content/home-story-image-sources.json.
const storyImages:Record<string,string>={
 'w-ucode':'/images/home/stories/ucode-v1.jpg',
 'w-uz-hk':'/images/home/stories/hong-kong-fintech-v2.jpg',
 'w-kenya-ai':'/images/home/stories/kenya-ai-v1.jpg',
 'w-opin':'/images/home/stories/opin-summit-v1.jpg',
 'w-tokyo-humanoids':'/images/home/stories/tokyo-humanoids-v1.jpg',
 'demo-creative-teams':'/images/home/stories/demo-project-v1.png',
}

export function homeStoryImage(article:Article):string|undefined{
 const original=article.media?.url
 const mapPlaceholder=!!original&&/\/maps\/[^/?]+\.svg(?:\?|$)/i.test(original)
 if(original&&!mapPlaceholder)return original
 return Object.hasOwn(storyImages,article.slug)?storyImages[article.slug]:original
}
