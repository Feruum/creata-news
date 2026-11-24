import type { Article as ArticleDocument, Locale } from '@creata/contracts'
export type { Locale, LocalizedText as Localized, Category, Author, SourceRef, Opportunity, User, AuthState, List } from '@creata/contracts'
export type Article = ArticleDocument & { readingMinutes?: Record<Locale, number> }
