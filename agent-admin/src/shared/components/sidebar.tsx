import { Link, useRouterState } from '@tanstack/react-router'
import { Files, Upload, Database } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

const navItems = [
  { to: '/documents', label: 'Documents', icon: Files },
  { to: '/upload', label: 'Upload', icon: Upload },
]

export function Sidebar() {
  const { location } = useRouterState()

  return (
    <aside className="flex w-60 flex-col border-r bg-card">
      <div className="flex h-14 items-center gap-2 border-b px-4">
        <Database className="h-5 w-5 text-primary" />
        <span className="font-semibold text-sm">Deimos Admin</span>
      </div>
      <nav className="flex-1 space-y-1 p-2">
        {navItems.map((item) => (
          <Link
            key={item.to}
            to={item.to}
            className={cn(
              'flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
              location.pathname === item.to
                ? 'bg-accent text-accent-foreground'
                : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground'
            )}
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </Link>
        ))}
      </nav>
    </aside>
  )
}
