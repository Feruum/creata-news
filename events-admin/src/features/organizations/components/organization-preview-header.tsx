import {
  Building2,
  Copy,
  LayoutPanelLeft,
  Maximize2,
  Minimize2,
  MoreHorizontal,
  PanelRight,
  RefreshCw,
  X
} from 'lucide-react'
import { toast } from 'sonner'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'
import type { OrganizationPreviewLayout } from '../types'

const layouts: { value: OrganizationPreviewLayout; label: string; icon: typeof PanelRight }[] = [
  { value: 'drawer', label: 'Drawer', icon: PanelRight },
  { value: 'full-page', label: 'Full Page', icon: Maximize2 },
  { value: 'pop-up', label: 'Pop Up', icon: LayoutPanelLeft },
  { value: 'minimize', label: 'Minimize', icon: Minimize2 }
]

export function OrganizationPreviewHeader({
  title,
  typeName,
  layout,
  onLayoutChange,
  onClose,
  onRefresh
}: {
  title: string
  typeName?: string
  layout: OrganizationPreviewLayout
  onLayoutChange: (layout: OrganizationPreviewLayout) => void
  onClose: () => void
  onRefresh: () => void
}) {
  return (
    <header className='club-preview-topbar'>
      {layout === 'minimize' ? (
        <button
          type='button'
          className='club-preview-restore'
          onClick={() => onLayoutChange('drawer')}
          aria-label={`Restore ${title} preview`}
        >
          <PanelRight size={16} />
          <span>{title}</span>
        </button>
      ) : (
        <nav aria-label='Club breadcrumb' className='club-preview-breadcrumb'>
          <span>Organizations</span>
          <span aria-hidden='true'>/</span>
          <strong title={title}>{title}</strong>
        </nav>
      )}
      <div className='club-preview-toolbar'>
        {typeName && layout !== 'minimize' && (
          <span className='club-preview-type'>
            <Building2 size={15} />
            {typeName}
          </span>
        )}
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger
            render={<button type='button' className='club-preview-icon-button' aria-label='Switch layout' />}
          >
            <PanelRight size={16} />
          </DropdownMenuTrigger>
          <DropdownMenuContent align='end' className='min-w-44'>
            <p className='px-2 py-1.5 text-xs text-muted-foreground'>Switch Layout</p>
            <DropdownMenuRadioGroup value={layout}>
              {layouts.map(({ value, label, icon: Icon }) => (
                <DropdownMenuRadioItem key={value} value={value} closeOnClick onClick={() => onLayoutChange(value)}>
                  <Icon size={15} />
                  {label}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        {layout !== 'minimize' && (
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger
              render={<button type='button' className='club-preview-icon-button' aria-label='More club actions' />}
            >
              <MoreHorizontal size={18} />
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end'>
              <DropdownMenuItem onClick={onRefresh}>
                <RefreshCw />
                Refresh club
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={async () => {
                  try {
                    await navigator.clipboard.writeText(title)
                    toast.success('Club name copied')
                  } catch {
                    toast.error('Could not copy club name')
                  }
                }}
              >
                <Copy />
                Copy club name
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onLayoutChange('minimize')}>
                <Minimize2 />
                Minimize preview
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
        <button type='button' className='club-preview-icon-button' aria-label='Close preview' onClick={onClose}>
          <X size={18} />
        </button>
      </div>
    </header>
  )
}
