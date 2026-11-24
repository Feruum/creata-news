import type { endpoints, StoryFollowUpInput } from '@creata/contracts'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAdmin } from './hooks'

export type ArticleFollowUpResult = endpoints['GET /api/creata/v1/admin/articles/{id}/follow-up']['response']
export type StoryFollowUpQueue = endpoints['GET /api/creata/v1/admin/story-follow-ups']['response']

export const storyFollowUpsPath = '/admin/story-follow-ups'
export const storyFollowUpsKey = ['story-follow-ups']
export const articleFollowUpPath = (articleId: string) => `/admin/articles/${encodeURIComponent(articleId)}/follow-up`
export const articleFollowUpKey = (articleId: string) => ['article-follow-up', articleId]

export function useStoryFollowUpMutation() {
  const { api, t, toast } = useAdmin()
  const client = useQueryClient()

  return useMutation({
    mutationFn: async ({ articleId, input }: { articleId: string; input: StoryFollowUpInput | null }) => {
      const path = articleFollowUpPath(articleId)
      if (input) return api<endpoints['PUT /api/creata/v1/admin/articles/{id}/follow-up']['response']>(path, { method: 'PUT', body: JSON.stringify(input) })
      await api(path, { method: 'DELETE' })
      return { item: null }
    },
    onSuccess: ({ item }, { articleId }) => {
      if (!item) {
        client.setQueryData<StoryFollowUpQueue>(storyFollowUpsKey, previous => {
          if (!previous) return previous
          const items = previous.items.filter(check => check.articleId !== articleId)
          return { ...previous, items, total: items.length }
        })
      }
      client.setQueryData<ArticleFollowUpResult>(articleFollowUpKey(articleId), previous => previous && { ...previous, item })
      void client.invalidateQueries({ queryKey: articleFollowUpKey(articleId) })
      void client.invalidateQueries({ queryKey: storyFollowUpsKey })
      toast(t(item ? 'saved' : 'followUpCompleted'))
    },
  })
}
