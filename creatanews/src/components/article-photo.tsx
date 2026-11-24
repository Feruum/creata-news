import { safeArticlePhotoUrl, safeExternalUrl, type ArticlePhoto, type Locale } from '@creata/contracts'
import { copy } from '@/lib/i18n'
import { MediaImage } from './media-image'
import styles from './article-view.module.css'

export function ArticlePhotoFigure({ photo, locale, cover = false }: {
  photo: ArticlePhoto
  locale: Locale
  cover?: boolean
}) {
  const src = safeArticlePhotoUrl(photo.url)
  const source = safeExternalUrl(photo.sourceUrl)
  const license = safeExternalUrl(photo.licenseUrl)
  if (!src) return null

  return <figure className={`${styles.photo} ${cover ? 'article-cover' : styles.inlinePhoto}`}>
    <MediaImage src={src} alt={photo.caption} priority={cover} referrerPolicy='no-referrer'/>
    <figcaption>{photo.caption}{photo.credit && ` · ${photo.credit}`}
      {photo.rights && <> · {license ? <a href={license} target='_blank' rel='noopener noreferrer'>{photo.rights}</a> : photo.rights}</>}
      {source && <> · <a href={source} target='_blank' rel='noopener noreferrer'>{copy[locale].photoSource}</a></>}
    </figcaption>
  </figure>
}
