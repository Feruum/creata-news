import { useQuery } from '@connectrpc/connect-query'
import { globalSearch } from '@studyverse/proto'
import { transport } from '@/shared/lib/api'

export function useSearchResults(query: string) {
  return useQuery(
    globalSearch,
    { query, limit: 10 },
    {
      transport,
      enabled: query.length >= 2,
      staleTime: 1000 * 60
    }
  )
}
