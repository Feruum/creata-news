import { useAuth } from '@/shared/lib/auth'
import { LogOut } from 'lucide-react'

export function Header() {
  const { session, logout } = useAuth()
  const email = (session?.identity?.traits as Record<string, unknown>)?.email as string | undefined

  return (
    <header className="flex h-14 items-center justify-between border-b px-6">
      <h1 className="text-lg font-semibold">Knowledge Base</h1>
      <div className="flex items-center gap-3">
        {email && <span className="text-sm text-muted-foreground">{email}</span>}
        <button
          onClick={logout}
          className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
        >
          <LogOut className="h-4 w-4" />
          Logout
        </button>
      </div>
    </header>
  )
}
