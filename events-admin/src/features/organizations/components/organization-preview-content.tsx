import { Tabs } from '@base-ui/react/tabs'
import type { Organization, OrganizationMember } from '@studyverse/proto'
import { Clock, FileText, Link2, Network, Send, Users } from 'lucide-react'
import { Suspense } from 'react'
import { useOrganizationMembersQuery, useOrganizationRolesQuery, useOrganizationSubscribers } from '../api/queries'
import { organizationSocialUrl } from '../lib/social-url'
import type { OrganizationPreviewTab } from '../types'
import { OrgMembershipHistory } from './org-membership-history'
import { OrganizationActions } from './organization-actions'
import { OrganizationPreviewDetails } from './organization-preview-details'
import { OrganizationPreviewTeam } from './organization-preview-team'
import { OrganizationStructureChart, OrganizationStructureChartSkeleton } from './organization-structure-chart'

const tabs: { value: OrganizationPreviewTab; label: string; icon: typeof FileText }[] = [
  { value: 'details', label: 'Details', icon: FileText },
  { value: 'team', label: 'Team', icon: Users },
  { value: 'structure', label: 'Structure', icon: Network },
  { value: 'history', label: 'History', icon: Clock }
]

function MemberAvatars({ members, total }: { members: OrganizationMember[]; total: number }) {
  if (!total) return null
  return (
    <div className='club-preview-avatars' role='img' aria-label={`${total} club members`}>
      {members.slice(0, 5).map((member) => {
        const name =
          [member.firstName, member.lastName].filter(Boolean).join(' ') || member.username || `User ${member.userId}`
        return <span key={member.id} title={name} aria-hidden='true' />
      })}
      {total > 5 && <span>+{total - 5}</span>}
    </div>
  )
}

export function OrganizationPreviewContent({
  organization,
  typeName,
  tab,
  onTabChange,
  canEdit,
  canManageMembers,
  onEdit,
  onDeleted,
  onRefresh,
  refreshing
}: {
  organization: Organization
  typeName: string
  tab: OrganizationPreviewTab
  onTabChange: (tab: OrganizationPreviewTab) => void
  canEdit: boolean
  canManageMembers: boolean
  onEdit: () => void
  onDeleted: () => void
  onRefresh: () => void
  refreshing: boolean
}) {
  const team = useOrganizationMembersQuery({ organizationId: organization.id })
  const roles = useOrganizationRolesQuery(organization.id, organization.organizationTypeId)
  const subscribers = useOrganizationSubscribers(organization.id)
  const website = organizationSocialUrl(organization.website)
  const telegram = organizationSocialUrl(organization.telegramChannel, 'telegram')
  const members = team.data?.members ?? []
  const memberCount = team.data?.total ?? team.data?.members.length
  const teamError = team.isError || roles.isError
  const teamLoading = team.isPending || roles.isPending
  return (
    <Tabs.Root
      className='club-preview-body'
      value={tab}
      onValueChange={(value) => {
        if (value === 'details' || value === 'team' || value === 'structure' || value === 'history') onTabChange(value)
      }}
    >
      <section className='club-preview-summary'>
        <div className='club-preview-identity'>
          <div className='club-preview-logo'>
            {organization.imageUrl ? (
              <img
                src={organization.imageUrl}
                alt={`${organization.title} logo`}
                onError={(event) => {
                  event.currentTarget.style.display = 'none'
                }}
              />
            ) : (
              <span className='club-preview-logo-fallback'>{organization.title.slice(0, 1)}</span>
            )}
          </div>
          <div className='club-preview-title'>
            <h2>{organization.title}</h2>
            {website && (
              <a className='club-preview-website' href={website} target='_blank' rel='noopener noreferrer'>
                <Link2 size={16} />
                <span>{organization.website?.replace(/^https?:\/\//, '').replace(/\/$/, '')}</span>
              </a>
            )}
          </div>
          <MemberAvatars members={members} total={memberCount ?? 0} />
        </div>
        <div className='club-preview-actions'>
          <OrganizationActions organization={organization} canEdit={canEdit} onEdit={onEdit} onDeleted={onDeleted} />
          {telegram && (
            <a className='club-preview-action' href={telegram} target='_blank' rel='noopener noreferrer'>
              <Send size={16} />
              Telegram
            </a>
          )}
          <button type='button' className='club-preview-action' onClick={() => onTabChange('team')}>
            <Users size={16} />
            View team
          </button>
        </div>
      </section>
      <Tabs.List aria-label='Club sections' className='club-preview-tabs' activateOnFocus>
        {tabs.map(({ value, label, icon: Icon }) => (
          <Tabs.Tab key={value} value={value} id={`club-${organization.id}-tab-${value}`}>
            <Icon size={16} />
            {label}
          </Tabs.Tab>
        ))}
      </Tabs.List>
      <div className='club-preview-scroll'>
        {tabs.map(({ value }) => (
          <Tabs.Panel
            key={value}
            value={value}
            keepMounted
            id={`club-${organization.id}-panel-${value}`}
            aria-labelledby={`club-${organization.id}-tab-${value}`}
          >
            {value === 'details' ? (
              <OrganizationPreviewDetails
                organization={organization}
                typeName={typeName}
                roles={roles.data?.roleTemplates}
                memberCount={memberCount}
                subscribers={subscribers.data?.count}
                loading={teamLoading || subscribers.isPending}
                error={teamError || subscribers.isError}
                refreshing={refreshing || team.isFetching || roles.isFetching || subscribers.isFetching}
                onRefresh={onRefresh}
                onTeam={() => onTabChange('team')}
              />
            ) : value === 'history' ? (
              tab === value && (
                <div className='club-preview-history'>
                  <OrgMembershipHistory organizationId={organization.id} />
                </div>
              )
            ) : (
              tab === value &&
              (teamError ? (
                <div className='club-preview-empty-state' role='alert'>
                  <h3>Could not load team</h3>
                  <p>Please try again to load members and roles.</p>
                  <button className='club-preview-action' type='button' onClick={onRefresh}>
                    Try again
                  </button>
                </div>
              ) : teamLoading ? (
                <div className='club-preview-skeleton' role='status' aria-label='Loading team'>
                  <div />
                  <div />
                  <div />
                </div>
              ) : value === 'team' ? (
                <OrganizationPreviewTeam
                  organizationId={organization.id}
                  organizationTypeId={organization.organizationTypeId}
                  members={members}
                  roles={roles.data?.roleTemplates ?? []}
                  canManageMembers={canManageMembers}
                  canManageSettings={canEdit}
                />
              ) : (
                <div className='club-preview-structure'>
                  <Suspense fallback={<OrganizationStructureChartSkeleton />}>
                    <OrganizationStructureChart
                      organizationId={organization.id}
                      organizationTypeId={organization.organizationTypeId}
                      canManageMembers={canManageMembers}
                      canManageSettings={canEdit}
                    />
                  </Suspense>
                </div>
              ))
            )}
          </Tabs.Panel>
        ))}
      </div>
    </Tabs.Root>
  )
}
