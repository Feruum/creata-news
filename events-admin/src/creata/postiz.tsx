import { useQuery } from '@tanstack/react-query'
import type { Article, List, Locale, MediaAsset, schemas } from '@creata/contracts'
import { adminDemo } from './demo'
import { useAdmin } from './hooks'
import { ErrorPanel, Loading } from './ui'

type Target = schemas['PostizTarget']
type Delivery = schemas['PostizDelivery']
const languages = { ru: 'RU', kk: 'ҚАЗ', en: 'EN' }
const copy = {
  channels: ['Соцсети через Postiz', 'Postiz арқылы әлеуметтік желілер', 'Social channels via Postiz'],
  language: ['Язык поста', 'Жазба тілі', 'Post language'],
  visibility: ['Доступ на YouTube', 'YouTube-та көру рұқсаты', 'YouTube visibility'],
  private: ['Только владельцу', 'Тек иесіне', 'Private'],
  unlisted: ['По ссылке', 'Сілтеме арқылы', 'Unlisted'],
  public: ['Для всех', 'Барлығына', 'Public'],
  attachment: ['Файл из медиатеки', 'Медиатекадағы файл', 'Media library asset'],
  cover: ['Медиа статьи', 'Мақаланың медиасы', 'Article media'],
  imageHint: ['JPEG до 8 МиБ с проверенными правами и общедоступным HTTPS-адресом.', 'Құқығы тексерілген, ашық HTTPS мекенжайы бар JPEG, 8 МиБ дейін.', 'Rights-reviewed JPEG, up to 8 MiB, at a public HTTPS URL.'],
  videoHint: ['MP4 с проверенными правами и общедоступным HTTPS-адресом.', 'Құқығы тексерілген, ашық HTTPS мекенжайы бар MP4.', 'Rights-reviewed MP4 at a public HTTPS URL.'],
  off: ['Postiz не подключён', 'Postiz қосылмаған', 'Postiz is not connected'],
  empty: ['Подключите аккаунт в Postiz', 'Postiz ішінде аккаунт қосыңыз', 'Connect an account in Postiz'],
  unavailable: ['Канал недоступен', 'Арна қолжетімсіз', 'Channel unavailable'],
  draftMode: ['Postiz сохранит выбранные посты как черновики.', 'Postiz таңдалған жазбаларды нобай ретінде сақтайды.', 'Postiz will save the selected posts as drafts.'],
  nowMode: ['После выхода статьи Postiz отправит посты в выбранные аккаунты.', 'Мақала жарияланған соң Postiz жазбаларды таңдалған аккаунттарға жібереді.', 'Postiz will send posts to the selected accounts once the article is published.'],
  deliveries: ['Публикации в соцсетях', 'Әлеуметтік желідегі жарияланымдар', 'Social posts'],
  keepScheduled: ['Ранее выбранные настройки публикации в соцсетях будут сохранены.', 'Әлеуметтік желілерде жариялаудың бұрын таңдалған баптаулары сақталады.', 'Previously selected social publishing settings will be kept.'],
  replaceScheduled: ['Изменить настройки соцсетей', 'Әлеуметтік желі баптауларын өзгерту', 'Change social publishing settings'],
  restoreScheduled: ['Сохранить предыдущие настройки', 'Алдыңғы баптауларды сақтау', 'Keep previous settings'],
  open: ['Открыть пост', 'Жазбаны ашу', 'Open post'],
  scheduled: ['Запланировано', 'Жоспарланған', 'Scheduled'],
  queued: ['В очереди', 'Кезекте', 'Queued'],
  sending: ['Отправляется', 'Жіберілуде', 'Sending'],
  accepted: ['Принято Postiz', 'Postiz қабылдады', 'Accepted by Postiz'],
  draft: ['Черновик в Postiz', 'Postiz ішіндегі нобай', 'Draft in Postiz'],
  published: ['Опубликовано', 'Жарияланды', 'Published'],
  failed: ['Ошибка доставки', 'Жеткізу қатесі', 'Delivery failed'],
  uncertain: ['Проверьте результат в Postiz', 'Нәтижені Postiz ішінен тексеріңіз', 'Check the result in Postiz'],
  cancelled: ['Отменено', 'Бас тартылды', 'Cancelled'],
  missing_media: ['Нужно медиа', 'Медиа қажет', 'Media required'],
} as const
function label(key: keyof typeof copy, locale: Locale) {
  return copy[key][locale === 'ru' ? 0 : locale === 'kk' ? 1 : 2]
}

function PostizPreview({ article, provider, locale }: { article: Article; provider: string; locale: Locale }) {
  const { api } = useAdmin()
  const preview = useQuery({
    queryKey: ['postiz-preview', article.id, article.revision, provider, locale],
    queryFn: () => api<schemas['PostizPreview']>(`/admin/articles/${encodeURIComponent(article.id)}/postiz/preview?${new URLSearchParams({provider, locale})}`),
  })
  if (preview.isPending) return <Loading/>
  if (preview.error) return <ErrorPanel error={preview.error}/>
  return <div className="callout" style={{ whiteSpace: 'pre-line', overflowWrap: 'anywhere' }}>{preview.data.content}</div>
}

function PostizMedia({ video, value, onChange }: { video: boolean; value?: string; onChange: (id?: string) => void }) {
  const { api, locale } = useAdmin()
  const library = useQuery({
    queryKey: ['postiz-media'],
    queryFn: () => api<List<MediaAsset>>('/admin/media'),
    staleTime: 30_000,
  })
  if (library.isPending) return <Loading/>
  if (library.error) return <ErrorPanel error={library.error}/>
  const assets = library.data.items.filter(asset => !asset.needsRightsReview && !!asset.credit && !!asset.rights &&
    (video ? asset.mime === 'video/mp4' : asset.mime === 'image/jpeg'))
  return <label>{label('attachment', locale)}
    <select value={value || ''} onChange={event => onChange(event.target.value || undefined)}>
      <option value="">{label('cover', locale)}</option>
      {assets.map(asset => <option key={asset.id} value={asset.id}>{asset.caption || asset.legacyPath || asset.id}</option>)}
    </select>
    <small>{label(video ? 'videoHint' : 'imageHint', locale)}</small>
  </label>
}

export function PublicationPostizOptions({ article, value, onChange }: {
  article: Article
  value: Target[] | undefined
  onChange: (targets: Target[] | undefined) => void
}) {
  const { locale } = useAdmin()
  if (adminDemo) return null
  if (article.status === 'scheduled' && value === undefined) return <div className="form-stack">
    <small>{label('keepScheduled', locale)}</small>
    <button className="button secondary" type="button" onClick={() => onChange([])}>{label('replaceScheduled', locale)}</button>
  </div>
  return <>
    {article.status === 'scheduled' && <button className="button secondary" type="button" onClick={() => onChange(undefined)}>{label('restoreScheduled', locale)}</button>}
    <PostizOptions article={article} value={value ?? []} onChange={onChange}/>
  </>
}

export function PostizOptions({ article, value, onChange }: {
  article: Article
  value: Target[]
  onChange: (targets: Target[]) => void
}) {
  const { api, locale } = useAdmin()
  const channels = useQuery({
    queryKey: ['postiz-channels'],
    queryFn: () => api<schemas['PostizChannels']>('/admin/postiz/channels'),
    enabled: !adminDemo,
    staleTime: 30_000,
    retry: false,
  })
  if (adminDemo) return null
  if (channels.isPending) return <Loading/>
  if (channels.error) return <ErrorPanel error={channels.error} retry={() => void channels.refetch()}/>
  if (!channels.data.configured) return <small className="muted">{label('off', locale)}</small>
  return <fieldset className="form-stack">
    <legend>{label('channels', locale)}</legend>
    <small>{label(channels.data.mode === 'draft' ? 'draftMode' : 'nowMode', locale)}</small>
    {!channels.data.items.length && <p>{label('empty', locale)}</p>}
    {channels.data.items.map(channel => {
      const target = value.find(item => item.channelId === channel.id)
      const others = value.filter(item => item.channelId !== channel.id)
      return <div key={channel.id} className="form-stack">
        <label className="checkbox-label"><input type="checkbox" checked={!!target} disabled={!channel.supported}
          onChange={event => onChange(event.target.checked ? [...others, { channelId: channel.id, locale }] : others)}/>
          {channel.name} ({channel.provider})</label>
        {!channel.supported && <small>{channel.reason || label('unavailable', locale)}</small>}
        {target && <>
          <label>{label('language', locale)} · {channel.name}
            <select value={target.locale} onChange={event => onChange([...others, { ...target, locale: event.target.value as Locale }])}>
              {Object.entries(languages).map(([id, name]) => <option key={id} value={id}>{name}</option>)}
            </select>
          </label>
          {['instagram', 'instagram-standalone', 'youtube'].includes(channel.provider) &&
            <PostizMedia video={channel.provider === 'youtube'} value={target.mediaId}
              onChange={mediaId => onChange([...others, { ...target, mediaId }])}/>}
          {channel.provider === 'youtube' && <label>{label('visibility', locale)}
            <select value={target.visibility || 'private'} onChange={event => onChange([...others, { ...target, visibility: event.target.value as Target['visibility'] }])}>
              {(['private','unlisted','public'] as const).map(mode => <option key={mode} value={mode}>{label(mode, locale)}</option>)}
            </select>
          </label>}
          <PostizPreview article={article} provider={channel.provider} locale={target.locale}/>
        </>}
      </div>
    })}
  </fieldset>
}

export function PostizDeliveries({ articleId }: { articleId: string }) {
  const { api, locale } = useAdmin()
  const deliveries = useQuery({
    queryKey: ['postiz-deliveries', articleId],
    queryFn: () => api<{ items: Delivery[] }>(`/admin/articles/${encodeURIComponent(articleId)}/postiz`),
    enabled: !adminDemo,
    refetchInterval: query => {
      const items = query.state.data?.items || []
      if (items.some(d => ['queued','sending','accepted'].includes(d.status))) return 4000
      return items.some(d => ['scheduled','uncertain','draft'].includes(d.status)) ? 60_000 : false
    },
  })
  if (adminDemo) return null
  if (deliveries.error) return <ErrorPanel error={deliveries.error} retry={() => void deliveries.refetch()}/>
  if (!deliveries.data?.items.length) return null
  return <section className="panel section-spacing" aria-label={label('deliveries', locale)}>
    <h2>{label('deliveries', locale)}</h2>
    <ul>{deliveries.data.items.map(d => <li key={d.id}>
      <strong>{d.channelName}</strong> · {languages[d.locale]} · v{d.revision} · {label(d.status, locale)}
      {d.error && <small className="source-error job-error-line">{d.error}</small>}
      {d.releaseUrl && /^https?:\/\//i.test(d.releaseUrl) && <> · <a href={d.releaseUrl} target="_blank" rel="noopener noreferrer">{label('open', locale)}</a></>}
    </li>)}</ul>
  </section>
}
