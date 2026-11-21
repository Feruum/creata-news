import { CheckInPhase, EventStatus } from '@studyverse/proto'
import { CopyIcon, ExternalLink, Loader2, LockIcon, QrCode } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { useEventCheckInInfo } from '../api/queries'
import { CheckInQrCode } from './check-in-qr-code'

interface EventCheckInPanelProps {
  eventId: number
  canManageRegistrations: boolean
  eventStatus: EventStatus
}

function getPhaseLabel(phase: CheckInPhase): string {
  switch (phase) {
    case CheckInPhase.LIVE:
      return 'Live'
    case CheckInPhase.GRACE:
      return 'Grace'
    case CheckInPhase.LOCKED:
      return 'Locked'
    case CheckInPhase.NONE:
      return 'Not Active'
    default:
      return 'Unknown'
  }
}

export function EventCheckInPanel({ eventId, canManageRegistrations, eventStatus }: EventCheckInPanelProps) {
  const [tokenCountdown, setTokenCountdown] = useState<number>(0)
  const enabled = canManageRegistrations && eventStatus === EventStatus.PUBLISHED

  const { data, isLoading, isError } = useEventCheckInInfo(eventId, enabled, 25000)
  const phase = data?.checkInPhase ?? CheckInPhase.UNSPECIFIED
  const isActive = phase === CheckInPhase.LIVE || phase === CheckInPhase.GRACE

  const checkInUrl = useMemo(() => {
    if (!data?.checkInUrl) return ''
    if (isActive && data.currentRotatingToken) {
      return `${data.checkInUrl}?code=${data.currentRotatingToken}`
    }
    return data.checkInUrl
  }, [data?.checkInUrl, data?.currentRotatingToken, isActive])

  useEffect(() => {
    if (!data?.tokenValidUntil || !isActive) {
      setTokenCountdown(0)
      return
    }

    const updateTokenCountdown = () => {
      const now = new Date()
      const validUntil = new Date(data.tokenValidUntil!)
      const diff = Math.max(0, Math.ceil((validUntil.getTime() - now.getTime()) / 1000))
      setTokenCountdown(diff)
    }

    updateTokenCountdown()
    const interval = setInterval(updateTokenCountdown, 1000)

    return () => clearInterval(interval)
  }, [data?.tokenValidUntil, isActive])

  const handleCopyUrl = useCallback(async () => {
    if (!checkInUrl) return
    try {
      await navigator.clipboard.writeText(checkInUrl)
      toast.success('Check-in URL copied')
    } catch {
      toast.error('Failed to copy check-in URL')
    }
  }, [checkInUrl])

  if (!canManageRegistrations) return null

  return (
    <div className='mt-6 rounded-lg border bg-muted/20 p-4'>
      <div className='flex items-center justify-between gap-3'>
        <div className='flex items-center gap-2'>
          <QrCode className='h-4 w-4 text-muted-foreground' />
          <h3 className='text-sm font-semibold'>Check-in Access</h3>
        </div>
        <Badge variant='outline'>{getPhaseLabel(phase)}</Badge>
      </div>

      {eventStatus !== EventStatus.PUBLISHED && (
        <p className='mt-3 text-xs text-muted-foreground'>Check-in becomes available after the event is published.</p>
      )}

      {enabled && isLoading && (
        <div className='mt-3 flex items-center gap-2 text-xs text-muted-foreground'>
          <Loader2 className='h-3.5 w-3.5 animate-spin' />
          Loading live check-in info...
        </div>
      )}

      {enabled && isError && (
        <p className='mt-3 text-xs text-amber-600 dark:text-amber-400'>
          Could not load check-in info with current permissions.
        </p>
      )}

      {enabled && !isLoading && !isError && isActive && (
        <div className='mt-3 space-y-2'>
          {phase === CheckInPhase.GRACE && (
            <p className='text-xs text-amber-600 dark:text-amber-400'>Event is in grace period.</p>
          )}
          {checkInUrl && <CheckInQrCode value={checkInUrl} />}
          <div className='flex items-center gap-2 rounded-md border bg-background/70 p-2'>
            <code className='flex-1 truncate text-xs'>{checkInUrl}</code>
            <Button variant='ghost' size='icon' className='h-7 w-7' onClick={handleCopyUrl}>
              <CopyIcon className='h-3.5 w-3.5' />
            </Button>
            <Button
              variant='ghost'
              size='icon'
              className='h-7 w-7'
              disabled={!checkInUrl}
              onClick={() => window.open(checkInUrl, '_blank', 'noopener,noreferrer')}
            >
              <ExternalLink className='h-3.5 w-3.5' />
            </Button>
          </div>
          <p className='text-xs text-muted-foreground'>QR token refreshes every {tokenCountdown || 0}s.</p>
        </div>
      )}

      {enabled && !isLoading && !isError && (phase === CheckInPhase.NONE || phase === CheckInPhase.LOCKED) && (
        <div className='mt-3 flex items-center gap-2 text-xs text-muted-foreground'>
          <LockIcon className='h-3.5 w-3.5' />
          <span>
            {phase === CheckInPhase.NONE
              ? 'Check-in opens 30 minutes before event start.'
              : 'Check-in period has ended for this event.'}
          </span>
        </div>
      )}
    </div>
  )
}
