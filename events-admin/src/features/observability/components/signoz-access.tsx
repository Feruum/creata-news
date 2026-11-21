import { Activity, ExternalLink } from 'lucide-react'
import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { useOfflineState } from '@/shared/hooks/use-offline-state'
import { useAuth } from '@/shared/lib/auth'
import { browserIsOnline, onPrivateCacheInvalidated } from '@/shared/lib/offline-runtime'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { Button, buttonVariants } from '@/shared/ui/components/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/shared/ui/components/dialog'
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem
} from '@/shared/ui/components/sidebar'
import { loadSigNozAccess, type SigNozInfo } from '../api/signoz-access'

type AccessState = { status: 'closed' | 'loading' | 'error' } | { status: 'ready'; owner: string; info: SigNozInfo }

export function SigNozAccess() {
  const { session, isAuthenticated } = useAuth()
  const { isAdmin } = useCurrentUser()
  const offline = useOfflineState()
  const [, updateExpiry] = useState(0)
  const expiresAt = Date.parse(session?.expires_at ?? '')

  useEffect(() => {
    if (!Number.isFinite(expiresAt)) return
    let timer: ReturnType<typeof setTimeout>
    const scheduleExpiry = () => {
      const remaining = expiresAt - Date.now()
      if (remaining <= 0) updateExpiry((value) => value + 1)
      else timer = setTimeout(scheduleExpiry, Math.min(remaining, 2_147_483_647))
    }
    scheduleExpiry()
    return () => clearTimeout(timer)
  }, [expiresAt])

  if (
    !isAuthenticated ||
    !isAdmin ||
    !session?.active ||
    !session.id ||
    !session.identity?.id ||
    offline.mode !== 'online' ||
    !browserIsOnline() ||
    !Number.isFinite(expiresAt) ||
    expiresAt <= Date.now()
  ) {
    return null
  }

  const owner = JSON.stringify([session.id, session.identity.id])
  // Remount before rendering whenever the verified session owner changes.
  return <SigNozAccessDialog key={owner} owner={owner} expiresAt={expiresAt} />
}

function SigNozAccessDialog({ owner, expiresAt }: { owner: string; expiresAt: number }) {
  const [open, setOpen] = useState(false)
  const [state, setState] = useState<AccessState>({ status: 'closed' })
  const [revealed, setRevealed] = useState(false)
  const emailId = useId()
  const passwordId = useId()
  const lifecycle = useRef({ owner, generation: 0, open: false, mounted: true })
  const controller = useRef<AbortController | null>(null)

  const discard = useCallback(() => {
    lifecycle.current.generation += 1
    lifecycle.current.open = false
    controller.current?.abort()
    controller.current = null
    setOpen(false)
    setState({ status: 'closed' })
    setRevealed(false)
  }, [])

  useEffect(() => {
    lifecycle.current.mounted = true
    const unsubscribe = onPrivateCacheInvalidated(discard)
    window.addEventListener('offline', discard)
    return () => {
      unsubscribe()
      window.removeEventListener('offline', discard)
      lifecycle.current.mounted = false
      lifecycle.current.open = false
      lifecycle.current.generation += 1
      controller.current?.abort()
      controller.current = null
    }
  }, [discard])

  const changeOpen = (next: boolean) => {
    if (!next) {
      discard()
      return
    }
    if (lifecycle.current.open || !browserIsOnline() || expiresAt <= Date.now()) return
    discard()
    const generation = lifecycle.current.generation
    const request = new AbortController()
    controller.current = request
    lifecycle.current.open = true
    setOpen(true)
    setState({ status: 'loading' })
    const isCurrent = () =>
      lifecycle.current.mounted &&
      lifecycle.current.open &&
      lifecycle.current.owner === owner &&
      lifecycle.current.generation === generation &&
      !request.signal.aborted &&
      browserIsOnline() &&
      expiresAt > Date.now()

    void loadSigNozAccess(request.signal)
      .then((info) => {
        if (isCurrent()) setState({ status: 'ready', owner, info })
      })
      .catch(() => {
        if (isCurrent()) setState({ status: 'error' })
      })
      .finally(() => {
        if (controller.current === request) controller.current = null
      })
  }

  const info = open && state.status === 'ready' && state.owner === owner ? state.info : null

  return (
    <SidebarGroup>
      <SidebarGroupContent>
        <SidebarMenu>
          <SidebarMenuItem>
            <Dialog open={open} onOpenChange={changeOpen}>
              <DialogTrigger render={<SidebarMenuButton tooltip='SigNoz' />}>
                <Activity />
                <span>SigNoz</span>
              </DialogTrigger>
              <DialogContent className='sm:max-w-md'>
                <DialogHeader>
                  <DialogTitle>SigNoz</DialogTitle>
                  <DialogDescription>Open SigNoz to view application metrics, logs and traces.</DialogDescription>
                </DialogHeader>
                {open && state.status === 'loading' && (
                  <p role='status' className='text-sm text-muted-foreground'>
                    Loading SigNoz access...
                  </p>
                )}
                {open && state.status === 'error' && (
                  <p role='alert' className='text-sm text-destructive'>
                    Unable to load SigNoz access. Close this dialog and try again.
                  </p>
                )}
                {info && !info.configured && <p className='text-sm text-muted-foreground'>SigNoz is unavailable.</p>}
                {info?.configured && (
                  <div className='space-y-4'>
                    {info.login ? (
                      <div className='space-y-3'>
                        <p className='text-sm text-muted-foreground'>Use these login details to sign in to SigNoz.</p>
                        <div className='space-y-2'>
                          <Label htmlFor={emailId}>SigNoz email</Label>
                          <Input
                            id={emailId}
                            type={revealed ? 'text' : 'password'}
                            value={info.login.email}
                            readOnly
                            autoComplete='off'
                            spellCheck={false}
                          />
                        </div>
                        <div className='space-y-2'>
                          <Label htmlFor={passwordId}>SigNoz password</Label>
                          <Input
                            id={passwordId}
                            type={revealed ? 'text' : 'password'}
                            value={info.login.password}
                            readOnly
                            autoComplete='off'
                            spellCheck={false}
                          />
                        </div>
                        <Button
                          variant='outline'
                          onClick={() => setRevealed((value) => !value)}
                          aria-pressed={revealed}
                        >
                          {revealed ? 'Hide login details' : 'Show login details'}
                        </Button>
                      </div>
                    ) : (
                      <p className='text-sm text-muted-foreground'>Sign in with your SigNoz account.</p>
                    )}
                    <a href={info.url} target='_blank' rel='noopener noreferrer' className={buttonVariants()}>
                      Open SigNoz
                      <ExternalLink />
                    </a>
                  </div>
                )}
              </DialogContent>
            </Dialog>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
