import type { OrganizationMember, RoleTemplate } from '@studyverse/proto'
import { Users } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useOrganizationMembers, useRoleTemplatesByOrg } from '@/features/organizations'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'

import { Avatar, AvatarFallback, AvatarImage } from '@/shared/ui/components/avatar'
import { Badge } from '@/shared/ui/components/badge'
import { Skeleton } from '@/shared/ui/components/skeleton'

import { AddMemberDialog } from './add-member-dialog'
import { MemberManagement } from './member-management'
import { RoleTemplateEditor } from './role-template-editor'

// ============================================================================
// TYPES
// ============================================================================

interface RoleNode {
  role: RoleTemplate
  members: OrganizationMember[]
  children: RoleNode[]
  // Layout coordinates for the role group
  x: number
  y: number
  width: number
  height: number
  // Individual member positions within the group
  memberPositions: { member: OrganizationMember; x: number; y: number }[]
}

interface OrganizationStructureChartProps {
  organizationId: number
  organizationTypeId: number
  canManageMembers?: boolean
  canManageSettings?: boolean
}

// ============================================================================
// CONSTANTS
// ============================================================================

const NODE_WIDTH = 160
const NODE_HEIGHT = 160
const HORIZONTAL_GAP = 32
const VERTICAL_GAP = 80
const MEMBERS_GAP = 16

// ============================================================================
// MEMBER NODE COMPONENT
// ============================================================================

interface MemberCardProps {
  member: OrganizationMember
  role: RoleTemplate
  style?: React.CSSProperties
}

function MemberCard({ member, role, style }: MemberCardProps) {
  const displayName =
    member.firstName && member.lastName
      ? `${member.firstName} ${member.lastName}`
      : member.username || member.email || 'Unknown'

  const initials = displayName
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)

  return (
    <div
      className='absolute flex flex-col items-center gap-2 rounded-xl border bg-card p-4 shadow-sm transition-all hover:shadow-md hover:border-primary/30'
      style={style}
    >
      <Avatar className='h-12 w-12 border-2 border-background shadow'>
        <AvatarImage src={undefined} alt={displayName} />
        <AvatarFallback className='text-sm font-medium'>{initials}</AvatarFallback>
      </Avatar>
      <div className='flex flex-col items-center gap-1 text-center'>
        <span className='flex items-center justify-center gap-1 text-sm font-medium leading-tight'>
          {displayName}
          <AchievementNameBadge badge={member.selectedBadge} userId={member.userId} />
        </span>
        <Badge variant='secondary' className='text-xs'>
          {role.displayName}
        </Badge>
        {role.description && (
          <span className='text-[10px] text-muted-foreground leading-tight max-w-[140px] line-clamp-2'>
            {role.description}
          </span>
        )}
      </div>
    </div>
  )
}

// ============================================================================
// SVG CONNECTION LINES
// ============================================================================

interface ConnectionLinesProps {
  roleNodes: RoleNode[]
  padding: number
}

function ConnectionLines({ roleNodes, padding }: ConnectionLinesProps) {
  const paths: React.ReactNode[] = []

  function collectPaths(node: RoleNode, parentNode?: RoleNode) {
    // Draw lines from parent role members to this role's members
    if (parentNode && parentNode.memberPositions.length > 0 && node.memberPositions.length > 0) {
      // Find center point of parent members
      const parentCenterX =
        parentNode.memberPositions.reduce((sum, p) => sum + p.x + NODE_WIDTH / 2, 0) / parentNode.memberPositions.length
      const parentBottomY = parentNode.memberPositions[0].y + NODE_HEIGHT

      // Draw lines to each child member
      for (const childPos of node.memberPositions) {
        const childCenterX = childPos.x + NODE_WIDTH / 2
        const childTopY = childPos.y

        // Create flowing curved path with vertical-horizontal-vertical pattern
        const midY = parentBottomY + (childTopY - parentBottomY) / 2

        const path = `
          M ${parentCenterX + padding} ${parentBottomY + padding}
          C ${parentCenterX + padding} ${midY + padding},
            ${childCenterX + padding} ${midY + padding},
            ${childCenterX + padding} ${childTopY + padding}
        `

        paths.push(
          <path
            key={`${parentNode.role.id}-${node.role.id}-${childPos.member.id}`}
            d={path}
            fill='none'
            stroke='currentColor'
            strokeWidth='2'
            strokeDasharray='8 6'
            strokeLinecap='round'
            className='text-muted-foreground/50 animate-flowing-dash'
          />
        )

        // Add small circle at the end
        paths.push(
          <circle
            key={`dot-${parentNode.role.id}-${node.role.id}-${childPos.member.id}`}
            cx={childCenterX + padding}
            cy={childTopY + padding - 2}
            r='4'
            fill='currentColor'
            className='text-muted-foreground/50'
          />
        )
      }
    }

    // Recurse to children
    for (const child of node.children) {
      collectPaths(child, node)
    }
  }

  // Start from root nodes
  for (const rootNode of roleNodes) {
    collectPaths(rootNode)
  }

  if (paths.length === 0) return null

  return (
    <svg
      className='pointer-events-none absolute inset-0 h-full w-full overflow-visible'
      aria-label='Organization structure connections'
    >
      <title>Organization structure connections</title>
      {paths}
    </svg>
  )
}

// ============================================================================
// LAYOUT HELPERS
// ============================================================================

function buildRoleHierarchy(members: OrganizationMember[], roles: RoleTemplate[]): RoleNode[] {
  const membersByRole = new Map<number, OrganizationMember[]>()

  for (const member of members) {
    const roleId = member.roleTemplateId
    if (!membersByRole.has(roleId)) {
      membersByRole.set(roleId, [])
    }
    membersByRole.get(roleId)!.push(member)
  }

  // Sort roles by sortOrder
  const sortedRoles = [...roles].sort((a, b) => a.sortOrder - b.sortOrder)

  // Find root roles (no parent) and build child map
  const rootRoles = sortedRoles.filter((r) => !r.parentTemplateId)
  const childRolesByParent = new Map<number, RoleTemplate[]>()

  for (const role of sortedRoles) {
    if (role.parentTemplateId) {
      if (!childRolesByParent.has(role.parentTemplateId)) {
        childRolesByParent.set(role.parentTemplateId, [])
      }
      childRolesByParent.get(role.parentTemplateId)!.push(role)
    }
  }

  // Build role tree recursively
  function buildRoleNode(role: RoleTemplate): RoleNode | null {
    const roleMembers = membersByRole.get(role.id) || []
    const childRoles = childRolesByParent.get(role.id) || []

    // Build child nodes first
    const childNodes: RoleNode[] = []
    for (const childRole of childRoles) {
      const childNode = buildRoleNode(childRole)
      if (childNode) {
        childNodes.push(childNode)
      }
    }

    // Only include this node if it has members OR has children with members
    if (roleMembers.length === 0 && childNodes.length === 0) {
      return null
    }

    return {
      role,
      members: roleMembers,
      children: childNodes,
      x: 0,
      y: 0,
      width: 0,
      height: 0,
      memberPositions: []
    }
  }

  // Build from all root roles
  const rootNodes: RoleNode[] = []
  for (const rootRole of rootRoles) {
    const node = buildRoleNode(rootRole)
    if (node) {
      rootNodes.push(node)
    }
  }

  return rootNodes
}

function layoutRoleTree(rootNodes: RoleNode[]): { width: number; height: number } {
  let maxWidth = 0
  let maxHeight = 0

  // Calculate width needed for a subtree
  function calculateSubtreeWidth(node: RoleNode): number {
    // Width for this node's members (side by side)
    const membersWidth =
      node.members.length > 0 ? node.members.length * NODE_WIDTH + (node.members.length - 1) * MEMBERS_GAP : 0

    // Width for children subtrees
    let childrenTotalWidth = 0
    for (const child of node.children) {
      childrenTotalWidth += calculateSubtreeWidth(child)
    }
    if (node.children.length > 1) {
      childrenTotalWidth += (node.children.length - 1) * HORIZONTAL_GAP
    }

    return Math.max(membersWidth, childrenTotalWidth, NODE_WIDTH)
  }

  // Layout a subtree starting at given position
  function layoutSubtree(node: RoleNode, x: number, y: number): number {
    const subtreeWidth = calculateSubtreeWidth(node)

    // Position members centered at this level
    const membersWidth =
      node.members.length > 0 ? node.members.length * NODE_WIDTH + (node.members.length - 1) * MEMBERS_GAP : 0

    const membersStartX = x + (subtreeWidth - membersWidth) / 2

    node.memberPositions = node.members.map((member, index) => ({
      member,
      x: membersStartX + index * (NODE_WIDTH + MEMBERS_GAP),
      y
    }))

    node.x = x
    node.y = y
    node.width = subtreeWidth
    node.height = node.members.length > 0 ? NODE_HEIGHT : 0

    // Update max dimensions
    for (const pos of node.memberPositions) {
      maxWidth = Math.max(maxWidth, pos.x + NODE_WIDTH)
      maxHeight = Math.max(maxHeight, pos.y + NODE_HEIGHT)
    }

    // Layout children
    if (node.children.length > 0) {
      const childY = y + (node.members.length > 0 ? NODE_HEIGHT + VERTICAL_GAP : 0)
      let childX = x

      for (const child of node.children) {
        const childWidth = layoutSubtree(child, childX, childY)
        childX += childWidth + HORIZONTAL_GAP
      }
    }

    return subtreeWidth
  }

  // Layout all root nodes side by side
  let currentX = 0
  for (const rootNode of rootNodes) {
    const width = layoutSubtree(rootNode, currentX, 0)
    currentX += width + HORIZONTAL_GAP * 2
  }

  return { width: maxWidth, height: maxHeight }
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export function OrganizationStructureChart({
  organizationId,
  organizationTypeId,
  canManageMembers = false,
  canManageSettings = false
}: OrganizationStructureChartProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [containerSize, setContainerSize] = useState({ width: 800, height: 600 })

  const { data: membersData } = useOrganizationMembers(organizationId)

  const { data: rolesData } = useRoleTemplatesByOrg({ organizationId, organizationTypeId })

  const members = membersData.members
  const roles = rolesData.roleTemplates
  const existingMemberUserIds = members.map((member: OrganizationMember) => member.userId)

  // Build and layout the role-based hierarchy
  const { roleNodes, layoutDimensions } = useMemo(() => {
    const hierarchy = buildRoleHierarchy(members, roles)
    const dimensions = layoutRoleTree(hierarchy)
    return { roleNodes: hierarchy, layoutDimensions: dimensions }
  }, [members, roles])

  // Update container size based on layout
  const PADDING = 60
  useEffect(() => {
    setContainerSize({
      width: Math.max(layoutDimensions.width + PADDING * 2, 400),
      height: Math.max(layoutDimensions.height + PADDING * 2, 300)
    })
  }, [layoutDimensions])

  // Flatten all member positions for rendering
  const allMemberPositions = useMemo(() => {
    const result: { member: OrganizationMember; role: RoleTemplate; x: number; y: number }[] = []

    function collect(node: RoleNode) {
      for (const pos of node.memberPositions) {
        result.push({
          member: pos.member,
          role: node.role,
          x: pos.x,
          y: pos.y
        })
      }
      for (const child of node.children) {
        collect(child)
      }
    }

    for (const node of roleNodes) {
      collect(node)
    }

    return result
  }, [roleNodes])

  if (members.length === 0) {
    return (
      <div className='flex flex-col items-center justify-center gap-4 py-12 text-muted-foreground'>
        <Users className='h-12 w-12 opacity-40' />
        <div className='text-center'>
          <p className='font-medium'>No members yet</p>
          <p className='text-sm'>Add members to see the organization structure</p>
        </div>
        {canManageSettings && (
          <RoleTemplateEditor
            organizationId={organizationId}
            organizationTypeId={organizationTypeId}
            roleTemplates={roles}
          />
        )}
        {canManageMembers && (
          <AddMemberDialog
            organizationId={organizationId}
            roleTemplates={roles}
            existingMemberUserIds={existingMemberUserIds}
          />
        )}
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      <div className='flex justify-end gap-2'>
        {canManageSettings && (
          <RoleTemplateEditor
            organizationId={organizationId}
            organizationTypeId={organizationTypeId}
            roleTemplates={roles}
          />
        )}
        {canManageMembers && (
          <AddMemberDialog
            organizationId={organizationId}
            roleTemplates={roles}
            existingMemberUserIds={existingMemberUserIds}
          />
        )}
      </div>
      {canManageMembers && <MemberManagement organizationId={organizationId} members={members} roles={roles} />}
      <div
        ref={containerRef}
        className='structure-chart-grid relative overflow-auto rounded-lg border bg-muted/5'
        style={{ minHeight: '400px' }}
      >
        <div
          className='relative'
          style={{
            width: containerSize.width,
            height: containerSize.height
          }}
        >
          <ConnectionLines roleNodes={roleNodes} padding={PADDING} />
          {allMemberPositions.map((pos) => (
            <MemberCard
              key={pos.member.id}
              member={pos.member}
              role={pos.role}
              style={{
                left: pos.x + PADDING,
                top: pos.y + PADDING,
                width: NODE_WIDTH
              }}
            />
          ))}
        </div>
      </div>
    </div>
  )
}

// ============================================================================
// LOADING SKELETON
// ============================================================================

export function OrganizationStructureChartSkeleton() {
  return (
    <div className='structure-chart-grid relative min-h-[400px] rounded-lg border bg-muted/5'>
      <div className='flex flex-col items-center gap-12 p-10'>
        {/* Top level - single leader */}
        <Skeleton className='h-[130px] w-[160px] rounded-xl' />
        {/* Dashed line placeholder */}
        <div className='h-6 w-px border-l-2 border-dashed border-muted-foreground/30' />
        {/* Second level - multiple VPs */}
        <div className='flex gap-8'>
          <Skeleton className='h-[130px] w-[160px] rounded-xl' />
          <Skeleton className='h-[130px] w-[160px] rounded-xl' />
        </div>
        {/* Dashed line placeholder */}
        <div className='h-6 w-px border-l-2 border-dashed border-muted-foreground/30' />
        {/* Third level - heads/members */}
        <div className='flex gap-6'>
          <Skeleton className='h-[130px] w-[160px] rounded-xl' />
          <Skeleton className='h-[130px] w-[160px] rounded-xl' />
          <Skeleton className='h-[130px] w-[160px] rounded-xl' />
        </div>
      </div>
    </div>
  )
}
