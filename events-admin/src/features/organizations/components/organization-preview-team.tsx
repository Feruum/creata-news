import type { OrganizationMember, RoleTemplate } from '@studyverse/proto'
import { Users } from 'lucide-react'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { AddMemberDialog } from './add-member-dialog'
import { MemberManagement } from './member-management'
import { RoleTemplateEditor } from './role-template-editor'

export function OrganizationPreviewTeam({
  organizationId,
  organizationTypeId,
  members,
  roles,
  canManageMembers,
  canManageSettings
}: {
  organizationId: number
  organizationTypeId: number
  members: OrganizationMember[]
  roles: RoleTemplate[]
  canManageMembers: boolean
  canManageSettings: boolean
}) {
  const { isAdmin, isStaff } = useCurrentUser()
  return (
    <section className='club-preview-team' aria-label='Club team'>
      <div className='club-preview-section-heading'>
        <h3>
          Team <span className='club-preview-count'>{members.length}</span>
        </h3>
        <div className='club-preview-team-actions'>
          {(canManageSettings || isAdmin || isStaff) && (
            <RoleTemplateEditor
              organizationId={organizationId}
              organizationTypeId={organizationTypeId}
              roleTemplates={roles}
              canManageSettings={canManageSettings}
            />
          )}
          {canManageMembers && (
            <AddMemberDialog
              organizationId={organizationId}
              roleTemplates={roles.filter((role) => role.isActive)}
              existingMemberUserIds={members.map((member) => member.userId)}
            />
          )}
        </div>
      </div>
      {members.length === 0 ? (
        <div className='club-preview-empty-state'>
          <Users size={28} />
          <h3>No members yet</h3>
          <p>Club members and their roles will appear here.</p>
        </div>
      ) : canManageMembers ? (
        <MemberManagement organizationId={organizationId} members={members} roles={roles} />
      ) : (
        <ul className='club-preview-member-list'>
          {members.map((member) => {
            const name =
              [member.firstName, member.lastName].filter(Boolean).join(' ') ||
              member.username ||
              `User ${member.userId}`
            return (
              <li key={member.id}>
                <span className='club-preview-avatar' aria-hidden='true'>
                  {name
                    .split(' ')
                    .map((part) => part[0])
                    .join('')
                    .slice(0, 2)}
                </span>
                <div>
                  <strong>{name}</strong>
                  <span>{roles.find((role) => role.id === member.roleTemplateId)?.displayName || 'Member'}</span>
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}
