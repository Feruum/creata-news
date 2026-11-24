import type { Locale, Localized } from '@/lib/models'
import { locales } from '@/lib/i18n'

export function launchText(value: Localized, locale: Locale) {
  const language = value[locale]?.trim() ? locale : locales.find(lang => value[lang]?.trim()) ?? locale
  return { language, text: value[language] || '' }
}

export const launchCopy = {
  ru: {title:'Запуски стартапов',intro:'Новые продукты и команды, которые их создают.',newest:'Новые запуски',product:'Продукт',team:'Команда',date:'Дата запуска',website:'Открыть продукт',back:'Все запуски',related:'Другие стартапы',empty:'Пока нет запусков',emptyText:'Новые проекты появятся здесь после публикации редакцией.',count:'проектов',query:'Название или идея',photo:'Изображение продукта'},
  kk: {title:'Стартаптар іске қосылуы',intro:'Жаңа өнімдер және оларды жасайтын командалар.',newest:'Жаңа жобалар',product:'Өнім',team:'Команда',date:'Іске қосылған күні',website:'Өнімді ашу',back:'Барлық жобалар',related:'Басқа стартаптар',empty:'Әзірге жобалар жоқ',emptyText:'Жаңа жобалар редакция жариялағаннан кейін осында пайда болады.',count:'жоба',query:'Атауы немесе идеясы',photo:'Өнім суреті'},
  en: {title:'Startup launches',intro:'New products and the teams bringing them to life.',newest:'New launches',product:'Product',team:'Team',date:'Launch date',website:'Visit product',back:'All launches',related:'More startups',empty:'No launches yet',emptyText:'New projects will appear here once published by the newsroom.',count:'projects',query:'Name or idea',photo:'Product image'},
} satisfies Record<Locale, Record<string,string>>

export const launchSocialCopy={
  ru:{vote:'Поддержать',unvote:'Снять голос',voteSignIn:'Войти и поддержать',discussion:'Обсуждение',writeComment:'Ваш комментарий',postComment:'Отправить',noComments:'Пока нет комментариев. Начните обсуждение.',signInDiscussion:'Войдите, чтобы присоединиться к обсуждению',removeComment:'Удалить',removeConfirm:'Удалить комментарий?',loading:'Загрузка обсуждения…',logo:'Логотип',tags:'Теги',sort:'Порядок',popular:'По голосам',newest:'Сначала новые',date:'Дата запуска'},
  kk:{vote:'Қолдау',unvote:'Дауысты қайтару',voteSignIn:'Кіріп, қолдау',discussion:'Талқылау',writeComment:'Пікіріңіз',postComment:'Жіберу',noComments:'Әзірге пікір жоқ. Талқылауды бастаңыз.',signInDiscussion:'Талқылауға қатысу үшін кіріңіз',removeComment:'Жою',removeConfirm:'Пікірді жою керек пе?',loading:'Талқылау жүктелуде…',logo:'Логотип',tags:'Тегтер',sort:'Реті',popular:'Дауыс саны бойынша',newest:'Алдымен жаңалары',date:'Іске қосылған күні'},
  en:{vote:'Upvote',unvote:'Remove vote',voteSignIn:'Sign in to upvote',discussion:'Discussion',writeComment:'Your comment',postComment:'Post comment',noComments:'No comments yet. Start the discussion.',signInDiscussion:'Sign in to join the discussion',removeComment:'Delete',removeConfirm:'Delete this comment?',loading:'Loading discussion…',logo:'Logo',tags:'Tags',sort:'Sort',popular:'Most upvoted',newest:'Newest first',date:'Launch date'},
} satisfies Record<Locale,Record<string,string>>

export const submissionCopy = {
  ru: {
    title: 'Мои заявки на запуск', add: 'Подать запуск', edit: 'Редактировать', unnamed: 'Черновик без названия',
    empty: 'У вас пока нет заявок.', name: 'Название продукта', tagline: 'Кратко о продукте', description: 'Описание продукта',
    team: 'Команда', website: 'Ссылка на продукт', photo: 'Ссылка на изображение', date: 'Дата запуска',
    translationHint: 'Заполните краткое описание, описание продукта и команду хотя бы на одном языке.',
    save: 'Сохранить черновик', submit: 'Отправить на модерацию', open: 'Открыть публикацию',
    draft: 'Черновик', in_review: 'На модерации', changes_requested: 'Нужны правки', published: 'Опубликовано', unpublished: 'Снято с публикации',
    reviewNote: 'Комментарий редактора', submitted: 'Заявка отправлена в редакцию.',
    incomplete: 'Укажите название, ссылку, дату запуска и заполните описание и команду. Сохраните черновик и отправьте снова.',
    conflict: 'Заявка уже изменилась. Обновите список перед повторной попыткой.', retry: 'Обновить заявки',
  },
  kk: {
    title: 'Іске қосу өтінімдерім', add: 'Өтінім беру', edit: 'Өңдеу', unnamed: 'Атаусыз жоба',
    empty: 'Әзірге өтінімдеріңіз жоқ.', name: 'Өнім атауы', tagline: 'Өнім туралы қысқаша', description: 'Өнім сипаттамасы',
    team: 'Команда', website: 'Өнімге сілтеме', photo: 'Суретке сілтеме', date: 'Іске қосу күні',
    translationHint: 'Өнім туралы қысқаша ақпаратты, сипаттаманы және команда туралы мәліметті кемінде бір тілде жазыңыз.',
    save: 'Жобаны сақтау', submit: 'Модерацияға жіберу', open: 'Жарияланымды ашу',
    draft: 'Жоба', in_review: 'Модерацияда', changes_requested: 'Түзету қажет', published: 'Жарияланды', unpublished: 'Жарияланымнан алынды',
    reviewNote: 'Редактор пікірі', submitted: 'Өтінім редакцияға жіберілді.',
    incomplete: 'Өнім атауын, сілтемесін және іске қосу күнін көрсетіп, өнім мен команданы сипаттаңыз. Жобаны сақтап, қайта жіберіңіз.',
    conflict: 'Өтінім өзгерді. Қайталап көрмес бұрын тізімді жаңартыңыз.', retry: 'Өтінімдерді жаңарту',
  },
  en: {
    title: 'My launch submissions', add: 'Submit a launch', edit: 'Edit', unnamed: 'Untitled draft',
    empty: 'You have no submissions yet.', name: 'Product name', tagline: 'Product tagline', description: 'Product description',
    team: 'Team', website: 'Product URL', photo: 'Image URL', date: 'Launch date',
    translationHint: 'Enter a tagline, product description, and team details in at least one language.',
    save: 'Save draft', submit: 'Send for review', open: 'View published launch',
    draft: 'Draft', in_review: 'In review', changes_requested: 'Changes requested', published: 'Published', unpublished: 'Unpublished',
    reviewNote: 'Editor’s feedback', submitted: 'Your launch was sent for review.',
    incomplete: 'Enter a product name, URL, launch date, tagline, description, and team details. Save the draft and try again.',
    conflict: 'This submission has changed. Refresh your submissions before trying again.', retry: 'Refresh submissions',
  },
} satisfies Record<Locale, Record<string, string>>
