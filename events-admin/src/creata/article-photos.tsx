import { safeArticlePhotoUrl, safeExternalUrl, type ArticlePhoto, type Media } from '@creata/contracts'
import { ImagePlus } from 'lucide-react'
import { tr } from './i18n'
import type { Locale } from './types'
import { Button, ExternalLink } from './ui'

export function ArticlePhotoFigure({ photo, language, priority = false }: {
  photo: ArticlePhoto
  language: Locale
  priority?: boolean
}) {
  const src = safeArticlePhotoUrl(photo.url)
  const source = safeExternalUrl(photo.sourceUrl)
  const license = safeExternalUrl(photo.licenseUrl)

  return <figure className='article-photo'>
    {src && <img src={src} alt={photo.caption} loading={priority ? 'eager' : 'lazy'} referrerPolicy='no-referrer'/>}
    <figcaption>{photo.caption}{photo.credit && ` · ${photo.credit}`}
      {photo.rights && <> · {license ? <ExternalLink href={license}>{photo.rights}</ExternalLink> : photo.rights}</>}
      {source && <> · <ExternalLink href={source}>{tr('photoSource', language)}</ExternalLink></>}
    </figcaption>
  </figure>
}

export function ArticleCoverPreview({ media, photo, language }: {
  media: Media
  photo?: ArticlePhoto
  language: Locale
}) {
  if (media.url) return <img className='cover-image' src={media.url} alt={media.caption}/>
  return photo
    ? <ArticlePhotoFigure photo={photo} language={language}/>
    : <div className='cover-placeholder'><ImagePlus size={30}/></div>
}

export function ArticlePhotosPanel({ photos, cover, language, disabled, onChange }: {
  photos: ArticlePhoto[]
  cover?: ArticlePhoto
  language: Locale
  disabled: boolean
  onChange: (photos: ArticlePhoto[]) => void
}) {
  if (photos.length === 0) return null
  const removeLabel = tr('removePhoto', language)

  return <section className='panel'>
    <h2>{tr('hermesPhotos', language)}</h2>
    <ul className='article-photo-list'>
      {photos.map((photo, index) => <li key={photo.url}>
        {photo === cover
          ? <p className='muted'>{`${tr('cover', language)} · ${photo.caption}`}</p>
          : <ArticlePhotoFigure photo={photo} language={language}/>}
        <Button type='button' variant='secondary' className='small' disabled={disabled}
          aria-label={`${removeLabel} · ${photo.caption}`}
          onClick={() => onChange(photos.filter((_, photoIndex) => photoIndex !== index))}>{removeLabel}</Button>
      </li>)}
    </ul>
  </section>
}
