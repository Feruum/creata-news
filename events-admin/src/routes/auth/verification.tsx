import type { VerificationFlow } from '@ory/client'
import { createFileRoute } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { z } from 'zod'
import { useAuth } from '@/shared/lib/auth'
import { getUserEmail } from '@/shared/lib/auth-utils'
import { kratos, startEmailVerification, verificationSubmitUrl } from '@/shared/lib/kratos'
import { Button, buttonVariants } from '@/shared/ui/components/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/components/card'
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'

export const Route = createFileRoute('/auth/verification')({
  validateSearch: z.object({ flow: z.string().optional() }),
  component: VerificationPage
})

function VerificationPage() {
  const { flow } = Route.useSearch()
  return (
    <main className='grid min-h-svh place-items-center p-4'>
      <Card className='w-full max-w-md'>
        <CardHeader>
          <CardTitle>Verify your email</CardTitle>
        </CardHeader>
        <CardContent className='space-y-4'>
          {flow ? <VerificationRequest key={flow} id={flow} /> : <StartVerification />}
        </CardContent>
      </Card>
    </main>
  )
}

function StartVerification() {
  return (
    <a href={startEmailVerification()} className={buttonVariants()}>
      Start verification
    </a>
  )
}

function inputValue(flow: VerificationFlow, name: string) {
  const input = flow.ui.nodes.find(
    (node) => node.attributes.node_type === 'input' && node.attributes.name === name
  )?.attributes
  return input?.node_type === 'input' && typeof input.value === 'string' ? input.value : ''
}

function VerificationRequest({ id }: { id: string }) {
  const { session } = useAuth()
  const [flow, setFlow] = useState<VerificationFlow | null>()
  useEffect(() => {
    const controller = new AbortController()
    void kratos.getVerificationFlow({ id }, { signal: controller.signal }).then(
      ({ data }) => {
        if (!controller.signal.aborted) setFlow(data.id === id && data.type === 'browser' ? data : null)
      },
      () => {
        if (!controller.signal.aborted) setFlow(null)
      }
    )
    return () => controller.abort()
  }, [id])

  if (flow === undefined) return <p role='status'>Loading verification…</p>
  if (flow?.state === 'passed_challenge') {
    const admin = new URL(import.meta.env.BASE_URL, window.location.origin)
    const hermes = new URL('hermes', admin).href
    const returnToHermes = flow.return_to === hermes
    return (
      <>
        <p>Your email is verified.</p>
        <a href={returnToHermes ? hermes : admin.href} className={buttonVariants()}>
          {returnToHermes ? 'Return to Hermes' : 'Return to administration'}
        </a>
      </>
    )
  }
  const csrf = flow && inputValue(flow, 'csrf_token')
  if (!flow || !csrf || !['choose_method', 'sent_email'].includes(flow.state)) {
    return (
      <>
        <p role='alert'>This verification request is unavailable or has expired.</p>
        <StartVerification />
      </>
    )
  }

  const sent = flow.state === 'sent_email'
  const email = inputValue(flow, 'email') || getUserEmail(session)
  const errors = [...(flow.ui.messages ?? []), ...flow.ui.nodes.flatMap((node) => node.messages)].filter(
    (message) => message.type === 'error'
  )
  const action = verificationSubmitUrl(id)
  const hidden = (
    <>
      <input type='hidden' name='method' value='code' />
      <input type='hidden' name='csrf_token' value={csrf} />
    </>
  )
  return (
    <>
      {errors.length > 0 && <p role='alert'>{errors.map((error) => error.text).join(' ')}</p>}
      <p className='text-sm text-muted-foreground'>
        {sent ? 'Check your inbox for the latest verification code.' : 'Request a code for your account email.'}
      </p>
      <form action={action} method='post' className='space-y-3'>
        {hidden}
        {sent ? (
          <>
            <Label htmlFor='verification-code'>Verification code</Label>
            <Input id='verification-code' name='code' autoComplete='one-time-code' required />
          </>
        ) : (
          <>
            <Label htmlFor='verification-email'>Email address</Label>
            <Input
              id='verification-email'
              name='email'
              type='email'
              autoComplete='email'
              defaultValue={email}
              required
            />
          </>
        )}
        <Button type='submit'>{sent ? 'Verify code' : 'Send verification code'}</Button>
      </form>
      {sent && email && (
        <form action={action} method='post'>
          {hidden}
          <input type='hidden' name='email' value={email} />
          <Button type='submit' variant='outline'>
            Send a new code
          </Button>
        </form>
      )}
    </>
  )
}
