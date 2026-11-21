import { useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { z } from 'zod'
import { kratos } from '@/shared/lib/kratos'
import { Button } from '@/shared/ui/components/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'

const errorSearchSchema = z.object({
  id: z.string().optional()
})

export const Route = createFileRoute('/auth/error')({
  validateSearch: errorSearchSchema,
  component: ErrorPage
})

function ErrorPage() {
  const navigate = useNavigate()
  const { id } = Route.useSearch()

  const { data: errorData, isLoading } = useQuery({
    queryKey: ['auth-error', id],
    queryFn: async () => {
      if (!id) return null
      const { data } = await kratos.getFlowError({ id })
      return data
    },
    enabled: !!id,
    retry: false
  })

  const error = errorData?.error as { message?: string; status?: string } | undefined
  const errorMessage = error?.message || 'There was a problem signing you in. Please try again.'
  const errorStatus = error?.status || 'Authentication Error'

  return (
    <div className='flex min-h-screen items-center justify-center bg-background'>
      <Card className='w-full max-w-md'>
        <CardHeader className='text-center'>
          <CardTitle className='text-destructive'>{errorStatus}</CardTitle>
          <CardDescription className='whitespace-pre-wrap'>
            {isLoading ? 'Loading error details...' : errorMessage}
          </CardDescription>
        </CardHeader>
        <CardContent className='flex justify-center'>
          <Button onClick={() => navigate({ to: '/auth/login' })}>Back to Login</Button>
        </CardContent>
      </Card>
    </div>
  )
}
