import { useEffect } from 'react'
import { pageCategory } from '@/shared/lib/product-events'
import { router } from '@/shared/lib/router'
import { trackProductEvent } from '@/shared/lib/telemetry'

let lastPath: string | undefined

export function ProductAnalytics() {
  useEffect(() => {
    const record = () => {
      const path = router.state.resolvedLocation?.pathname ?? router.state.location.pathname
      if (lastPath === path) return
      lastPath = path
      trackProductEvent('page_view', { surface: 'admin', category: pageCategory(path) })
    }
    record()
    return router.subscribe('onResolved', record)
  }, [])
  return null
}
