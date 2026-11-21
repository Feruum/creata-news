import { createConnectQueryKey } from '@connectrpc/connect-query'
import { getOrganizationSubscriberCount } from '@studyverse/proto'
import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/shared/ui/components/alert-dialog'
import {
  organizationDetailKey,
  organizationMembershipHistoryKey,
  organizationMembersKey,
  organizationRoleTemplatesKey,
  organizationTypesListKey
} from '../api/keys'
import { useOrganizationPreview, useOrganizationTypesQuery } from '../api/queries'
import type { OrganizationPreviewLayout, OrganizationPreviewTab } from '../types'
import { OrganizationFormContent } from './create-organization-form'
import { OrganizationPreviewContent } from './organization-preview-content'
import { OrganizationPreviewHeader } from './organization-preview-header'
import { OrganizationPreviewShell } from './organization-preview-shell'

interface OrganizationInfoModalProps {
  organizationId: number
  children: React.ReactNode
}

export function OrganizationInfoModal({ organizationId, children }: OrganizationInfoModalProps) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button type='button' onClick={() => setOpen(true)} className='cursor-pointer text-left'>
        {children}
      </button>
      <OrganizationInfoModalControlled organizationId={open ? organizationId : null} onClose={() => setOpen(false)} />
    </>
  )
}

interface OrganizationInfoModalControlledProps {
  organizationId: number | null
  onClose: () => void
  onOrganizationChange?: (organizationId: number | null) => void
}

type PendingAction = { type: 'close' } | { type: 'switch'; id: number }

export function OrganizationInfoModalControlled({
  organizationId,
  onClose,
  onOrganizationChange
}: OrganizationInfoModalControlledProps) {
  const [activeId, setActiveId] = useState(organizationId)
  const [open, setOpen] = useState(organizationId !== null)
  const [layout, setLayout] = useState<OrganizationPreviewLayout>('drawer')
  const [tab, setTab] = useState<OrganizationPreviewTab>('details')
  const [editing, setEditing] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [pending, setPending] = useState<PendingAction | null>(null)
  const lastRequested = useRef(organizationId)
  const returnFocusRef = useRef<HTMLElement | null>(null)
  const queryClient = useQueryClient()
  const result = useOrganizationPreview(activeId ?? 0, open && activeId !== null)
  const types = useOrganizationTypesQuery(open)
  const organization = result.data?.organization
  const typeName =
    types.data?.organizationTypes.find((type) => type.id === organization?.organizationTypeId)?.title || 'Organization'

  useEffect(() => {
    if (lastRequested.current === organizationId) return
    lastRequested.current = organizationId
    if (submitting || dirty) {
      if (!submitting) setPending(organizationId === null ? { type: 'close' } : { type: 'switch', id: organizationId })
      lastRequested.current = activeId
      onOrganizationChange?.(activeId)
      return
    }
    if (organizationId === null) setOpen(false)
    else {
      const row = document.querySelector('tr[data-preview-active=true]')
      const focused = document.activeElement
      if (row instanceof HTMLElement) returnFocusRef.current = row
      else if (focused instanceof HTMLElement && !focused.closest('.club-preview, [role="alertdialog"]'))
        returnFocusRef.current = focused
      setActiveId(organizationId)
      setOpen(true)
      setEditing(false)
    }
  }, [organizationId, activeId, dirty, submitting, onOrganizationChange])

  const close = () => {
    if (submitting) return
    if (dirty) {
      setPending({ type: 'close' })
      return
    }
    const row = document.querySelector('tr[data-preview-active=true]')
    if (row instanceof HTMLElement) returnFocusRef.current = row
    setOpen(false)
    setEditing(false)
    onClose()
  }
  const refresh = () => {
    if (activeId === null) return
    for (const queryKey of [
      organizationDetailKey(activeId),
      organizationMembersKey(activeId),
      organizationRoleTemplatesKey(),
      organizationTypesListKey(),
      organizationMembershipHistoryKey(activeId),
      createConnectQueryKey({
        schema: getOrganizationSubscriberCount,
        input: { organizationId: activeId },
        cardinality: 'finite'
      })
    ])
      void queryClient.invalidateQueries({ queryKey })
  }
  const discard = () => {
    if (submitting) return
    setDirty(false)
    setEditing(false)
    if (pending?.type === 'switch') {
      setActiveId(pending.id)
      setOpen(true)
      onOrganizationChange?.(pending.id)
    } else {
      const row = document.querySelector('tr[data-preview-active=true]')
      if (row instanceof HTMLElement) returnFocusRef.current = row
      setOpen(false)
      onClose()
    }
    setPending(null)
  }

  if (activeId === null) return null
  return (
    <>
      <OrganizationPreviewShell open={open} layout={layout} onClose={close} returnFocusRef={returnFocusRef}>
        <OrganizationPreviewHeader
          title={organization?.title || 'Club details'}
          typeName={typeName}
          layout={layout}
          onLayoutChange={setLayout}
          onClose={close}
          onRefresh={refresh}
        />
        <div className='club-preview-body' hidden={layout === 'minimize'}>
          {!organization ? (
            result.isError ? (
              <div className='club-preview-empty-state' role='alert'>
                <h3>Could not load club</h3>
                <p>{result.error.message}</p>
                <button type='button' className='club-preview-action' onClick={() => result.refetch()}>
                  Try again
                </button>
              </div>
            ) : result.isPending ? (
              <div className='club-preview-skeleton' role='status' aria-label='Loading club'>
                <div />
                <div />
                <div />
                <div />
                <div />
              </div>
            ) : (
              <div className='club-preview-empty-state'>
                <h3>Club not found</h3>
                <p>This club may have been removed.</p>
              </div>
            )
          ) : editing && result.data?.canEdit ? (
            types.isPending && !types.data ? (
              <div className='club-preview-skeleton' role='status' aria-label='Loading organization settings'>
                <div />
                <div />
                <div />
              </div>
            ) : types.isError && !types.data ? (
              <div className='club-preview-empty-state' role='alert'>
                <h3>Could not load organization settings</h3>
                <button type='button' className='club-preview-action' onClick={() => types.refetch()}>
                  Try again
                </button>
              </div>
            ) : (
              <OrganizationFormContent
                key={activeId}
                organization={organization}
                onSaved={() => {
                  setDirty(false)
                  setSubmitting(false)
                  setEditing(false)
                  refresh()
                }}
                onCancel={() => {
                  setDirty(false)
                  setSubmitting(false)
                  setEditing(false)
                }}
                onDirtyChange={setDirty}
                onSubmittingChange={setSubmitting}
              />
            )
          ) : (
            <OrganizationPreviewContent
              key={activeId}
              organization={organization}
              typeName={typeName}
              tab={tab}
              onTabChange={setTab}
              canEdit={result.data?.canEdit ?? false}
              canManageMembers={result.data?.canManageMembers ?? false}
              onEdit={() => setEditing(true)}
              onDeleted={() => {
                setOpen(false)
                onClose()
              }}
              onRefresh={refresh}
              refreshing={result.isFetching}
            />
          )}
        </div>
      </OrganizationPreviewShell>
      <AlertDialog open={pending !== null} onOpenChange={(next) => !next && setPending(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
            <AlertDialogDescription>
              Your edits have not been saved. Keep editing or discard them to{' '}
              {pending?.type === 'switch' ? 'open the other club' : 'close the preview'}.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep editing</AlertDialogCancel>
            <AlertDialogAction onClick={discard} disabled={submitting}>
              Discard changes
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
