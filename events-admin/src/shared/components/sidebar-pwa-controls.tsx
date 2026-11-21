import { Download } from 'lucide-react'
import { useState } from 'react'
import { usePwa } from '@/shared/hooks/use-pwa'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from '@/shared/ui/components/sidebar'

export function SidebarPwaControls() {
  const { canInstall, isIos, isInstalled, install } = usePwa()
  const [instructions, setInstructions] = useState(false)
  if (isInstalled || (!canInstall && !isIos)) return null
  return (
    <>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            tooltip='Install AITU EMS'
            aria-label='Install AITU EMS'
            onClick={() => {
              if (canInstall) void install('sidebar')
              else setInstructions(true)
            }}
          >
            <Download />
            <span className='group-data-[collapsible=icon]:hidden'>Install AITU EMS</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
      <Dialog open={instructions} onOpenChange={setInstructions}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Install AITU EMS</DialogTitle>
            <DialogDescription>Open this page in Safari. Tap Share, then Add to Home Screen.</DialogDescription>
          </DialogHeader>
          <Button onClick={() => setInstructions(false)}>Got it</Button>
        </DialogContent>
      </Dialog>
    </>
  )
}
