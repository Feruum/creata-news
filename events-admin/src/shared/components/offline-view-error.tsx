import { Component, type ReactNode } from 'react'
import { isOfflineReadUnavailable } from '@/shared/lib/offline-runtime'
import { Button } from '@/shared/ui/components/button'

export class OfflineViewError extends Component<{ children: ReactNode; location: string }, { error: Error | null }> {
  state: { error: Error | null } = { error: null }
  static getDerivedStateFromError(error: Error) {
    return { error }
  }
  componentDidUpdate(previous: Readonly<{ children: ReactNode; location: string }>) {
    if (previous.location !== this.props.location && this.state.error) this.setState({ error: null })
  }
  render() {
    if (!this.state.error) return this.props.children
    if (!isOfflineReadUnavailable(this.state.error)) throw this.state.error
    return (
      <div role='alert' className='rounded-lg border p-6'>
        <h2 className='text-lg font-semibold'>This view is unavailable offline</h2>
        <p className='mt-2 text-sm text-muted-foreground'>{this.state.error.message}</p>
        <Button className='mt-4' variant='outline' onClick={() => this.setState({ error: null })}>
          Try again
        </Button>
      </div>
    )
  }
}
