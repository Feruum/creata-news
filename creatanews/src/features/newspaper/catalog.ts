import type { Locale } from '@/lib/models'

export type NewspaperIssue = {
  id: string
  number: string
  date: string
  pages: number
  language: Locale
  cover: string
  headline: string
  pdf?: string
  title?: string
}

export const newspaperCopy = {
  ru: {
    title: 'Газета', archive: 'Архив выпусков', issue: 'Выпуск',
    intro: 'Истории, люди и события Creata — на страницах газеты. Каждый номер собран в отдельный выпуск с фотографиями и статьями.',
    read: 'Читать выпуск', back: 'Все выпуски', pages: 'полос', language: 'Выпуск на русском', pdf: 'Открыть PDF',
    months: ['Январь','Февраль','Март','Апрель','Май','Июнь','Июль','Август','Сентябрь','Октябрь','Ноябрь','Декабрь'],
  },
  kk: {
    title: 'Газет', archive: 'Нөмірлер мұрағаты', issue: 'Нөмір',
    intro: 'Creata оқиғалары мен адамдары — газет беттерінде. Әр нөмір фотосуреттер мен мақалалардан тұратын жеке шығарылым.',
    read: 'Нөмірді оқу', back: 'Барлық нөмір', pages: 'бет', language: 'Орыс тіліндегі нөмір', pdf: 'PDF ашу',
    months: ['Қаңтар','Ақпан','Наурыз','Сәуір','Мамыр','Маусым','Шілде','Тамыз','Қыркүйек','Қазан','Қараша','Желтоқсан'],
  },
  en: {
    title: 'Newspaper', archive: 'Issue archive', issue: 'Issue',
    intro: 'Creata stories, people and events, in newspaper form. Each edition brings its own collection of photographs and articles.',
    read: 'Read issue', back: 'All issues', pages: 'pages', language: 'Russian-language issue', pdf: 'Open PDF',
    months: ['January','February','March','April','May','June','July','August','September','October','November','December'],
  },
} satisfies Record<Locale, object>

export function newspaperDate(date: string, locale: Locale) {
  const [year, month] = date.split('-')
  return `${newspaperCopy[locale].months[Number(month) - 1]} ${year}`
}
