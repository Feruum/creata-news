import { ExternalLink } from 'lucide-react'
import { Button, buttonVariants } from '@/shared/ui/components/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'
import { useHermesAccess } from '../hooks/use-hermes-access'

export function HermesLauncher() {
  const { state, reload } = useHermesAccess()

  return (
    <div className='w-full max-w-2xl space-y-6'>
      <div className='space-y-2'>
        <h1 className='text-2xl font-semibold tracking-tight'>Hermes</h1>
        <p className='text-sm text-muted-foreground'>Investigate incidents and configure your diagnostics assistant.</p>
      </div>
      {state.status === 'loading' && (
        <p role='status' className='text-sm text-muted-foreground'>
          Checking Hermes access...
        </p>
      )}
      {state.status === 'denied' && (
        <p role='alert' className='text-sm text-muted-foreground'>
          Hermes is available only to its owner.
        </p>
      )}
      {state.status === 'unavailable' && (
        <div className='space-y-3'>
          <p role='alert' className='text-sm text-muted-foreground'>
            Hermes is unavailable. Try again.
          </p>
          <Button variant='outline' onClick={reload}>
            Try again
          </Button>
        </div>
      )}
      {state.status === 'ready' && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Owner workspace</CardTitle>
              <CardDescription>
                Open Hermes to connect ChatGPT, set up integrations and review incidents.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <a href={state.info.dashboardUrl} target='_blank' rel='noopener noreferrer' className={buttonVariants()}>
                Open Hermes
                <ExternalLink aria-hidden='true' />
              </a>
            </CardContent>
          </Card>
          <div className='space-y-2 text-sm text-muted-foreground'>
            <p>Only your Studyverse account can configure Hermes.</p>
            <p>
              To use Telegram, open AITU Map, choose Open menu and link Telegram in your account card. Then connect it
              in Hermes. To connect a group, send /group to the bot in your private chat. Hermes answers your commands
              in that group and sends automatic reports to you privately.
            </p>
            <a
              href='https://map.webspace.cat/'
              target='_blank'
              rel='noopener noreferrer'
              className={buttonVariants({ variant: 'outline' })}
            >
              Open AITU Map
              <ExternalLink aria-hidden='true' />
            </a>
          </div>
        </>
      )}
    </div>
  )
}
