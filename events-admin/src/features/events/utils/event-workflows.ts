import { Code, ConnectError } from '@connectrpc/connect'

export function getSubmitIntent(submitter: { value?: string } | null): 'draft' | 'publish' {
  return submitter?.value === 'draft' ? 'draft' : 'publish'
}

export async function copyPublicEventLink(
  event: { id: number; slug: string },
  mapOrigin: string,
  clipboard: Pick<Clipboard, 'writeText'>
) {
  const url = new URL(`/event/${encodeURIComponent(event.slug || String(event.id))}`, mapOrigin).toString()
  await clipboard.writeText(url)
}

export async function finishCampusDraft({
  requestBooking,
  checkReadiness,
  publish
}: {
  requestBooking?: () => Promise<unknown>
  checkReadiness: () => Promise<{ canPublishNow: boolean }>
  publish: () => Promise<unknown>
}): Promise<'published' | 'pending' | 'booking-failed' | 'publish-failed'> {
  if (requestBooking) {
    try {
      await requestBooking()
    } catch (error) {
      if (!(error instanceof ConnectError && error.code === Code.AlreadyExists)) return 'booking-failed'
    }
  }
  try {
    if (!(await checkReadiness()).canPublishNow) return 'pending'
    await publish()
    return 'published'
  } catch {
    return 'publish-failed'
  }
}
