export type { Locale, LocalizedText, ArticleStatus, SourceRef, Media, Author, Category, Article, User, Story, Opportunity, Source, Job, Settings, Digest, List } from '@creata/contracts'
import type { Article, LocalizedText } from '@creata/contracts'
export type Mode = 'editorial' | 'hermes'
export type ArticleAction = 'submit' | 'approve' | 'publish' | 'schedule' | 'unpublish' | 'reject' | 'rework'
export type { AuthState as Session } from '@creata/contracts'
export interface Revision extends Pick<Article, 'actionGuide' | 'storyUpdates'> { revision: number; title: LocalizedText; lead: LocalizedText; body: LocalizedText; summary: LocalizedText; actions: LocalizedText; sources: import('@creata/contracts').SourceRef[]; reviewNotes: string[]; createdAt: string; actorId?: string; note?: string }
export interface AuditEntry { id: string; actorId: string; action: string; targetId: string; createdAt: string; detail?: unknown }
export interface Analytics { published: number; stories: number; subscribers: number; views: number; saves: number; opportunityClicks: number; statuses: Record<string, number>; daily: Array<{ date: string; views?: number; count?: number }>; aiCostUsd: number }
