import type { Organization, RoleTemplate } from '@studyverse/proto'
import { OrganizationStatus } from '@studyverse/proto'
import {
  Building2,
  CalendarDays,
  Clock,
  Globe,
  Instagram,
  Link2,
  type LucideIcon,
  MessageCircle,
  RefreshCw,
  Send,
  Users
} from 'lucide-react'
import type { ReactNode } from 'react'
import { PlateEditor } from '@/features/events'
import { isPlateValueEmpty, parsePlateJson } from '@/shared/lib/plate-utils'
import { organizationSocialUrl } from '../lib/social-url'

function Property({ icon: Icon, label, children }: { icon: LucideIcon; label: string; children: ReactNode }) {
  return (
    <div className='club-preview-property'>
      <dt>
        <Icon size={16} />
        <span>{label}</span>
      </dt>
      <dd>{children || <span className='club-preview-empty'>Not provided</span>}</dd>
    </div>
  )
}

function SocialValue({ value, platform }: { value?: string; platform?: string }) {
  if (!value?.trim()) return <span className='club-preview-empty'>Not provided</span>
  const url = organizationSocialUrl(value, platform)
  return url ? (
    <a href={url} target='_blank' rel='noopener noreferrer'>
      {value?.replace(/^https?:\/\//, '').replace(/\/$/, '')}
    </a>
  ) : (
    <span>{value}</span>
  )
}

function dateLabel(value: string) {
  if (!value || Number.isNaN(Date.parse(value))) return 'Not available'
  return new Intl.DateTimeFormat('en', { month: 'long', day: 'numeric', year: 'numeric' }).format(new Date(value))
}

export function OrganizationPreviewDetails({
  organization,
  typeName,
  roles,
  memberCount,
  subscribers,
  loading,
  error,
  refreshing,
  onRefresh,
  onTeam
}: {
  organization: Organization
  typeName: string
  roles: RoleTemplate[] | undefined
  memberCount: number | undefined
  subscribers: number | undefined
  loading: boolean
  error: boolean
  refreshing: boolean
  onRefresh: () => void
  onTeam: () => void
}) {
  const org = organization
  const status = {
    [OrganizationStatus.ACTIVE]: 'Active',
    [OrganizationStatus.FROZEN]: 'Frozen',
    [OrganizationStatus.ARCHIVED]: 'Archived',
    [OrganizationStatus.UNSPECIFIED]: 'Unknown'
  }[org.status]
  const links = [
    org.website,
    org.instagram,
    org.telegramChannel,
    org.telegramChat,
    org.youtube,
    org.tiktok,
    org.linkedin
  ].filter((value) => value?.trim()).length
  const description = parsePlateJson(org.description)
  const metrics = [
    { key: 'members', title: 'Members', value: memberCount, caption: 'People in this club', onClick: onTeam },
    {
      key: 'roles',
      title: 'Active roles',
      value: roles?.filter((role) => role.isActive).length,
      caption: 'Roles available in the club',
      onClick: onTeam
    },
    { key: 'subscribers', title: 'Subscribers', value: subscribers, caption: 'People following club updates' },
    { key: 'links', title: 'Connected links', value: links, caption: 'Website and social profiles' }
  ]

  return (
    <div className='club-preview-details'>
      <dl className='club-preview-properties'>
        <Property icon={Building2} label='Type'>
          {typeName}
        </Property>
        <Property icon={Clock} label='Status'>
          <span className='club-preview-status' data-status={status?.toLowerCase()}>
            {status}
          </span>
        </Property>
        <Property icon={Globe} label='Website'>
          <SocialValue value={org.website} />
        </Property>
        <Property icon={Send} label='Telegram'>
          <SocialValue value={org.telegramChannel} platform='telegram' />
        </Property>
        <Property icon={Instagram} label='Instagram'>
          <SocialValue value={org.instagram} platform='instagram' />
        </Property>
        {org.telegramChat && (
          <Property icon={MessageCircle} label='Telegram chat'>
            <SocialValue value={org.telegramChat} platform='telegram' />
          </Property>
        )}
        {org.linkedin && (
          <Property icon={Link2} label='LinkedIn'>
            <SocialValue value={org.linkedin} />
          </Property>
        )}
        {org.youtube && (
          <Property icon={Link2} label='YouTube'>
            <SocialValue value={org.youtube} />
          </Property>
        )}
        {org.tiktok && (
          <Property icon={Link2} label='TikTok'>
            <SocialValue value={org.tiktok} platform='tiktok' />
          </Property>
        )}
        <Property icon={Users} label='Members'>
          <button type='button' className='club-preview-text-button' onClick={onTeam}>
            {memberCount === undefined ? 'View team' : `${memberCount} ${memberCount === 1 ? 'member' : 'members'}`}
          </button>
        </Property>
        <Property icon={CalendarDays} label='Created'>
          {dateLabel(org.createdAt)}
        </Property>
        <Property icon={Clock} label='Updated'>
          {dateLabel(org.updatedAt)}
        </Property>
      </dl>
      {!isPlateValueEmpty(description) && (
        <section className='club-preview-about'>
          <h3>About</h3>
          <PlateEditor value={description} onChange={() => {}} readOnly />
        </section>
      )}
      <section className='club-preview-overview' aria-label='Club overview'>
        <div className='club-preview-section-heading'>
          <h3>Overview</h3>
          <button type='button' className='club-preview-text-button' onClick={onRefresh} disabled={refreshing}>
            <RefreshCw size={15} className={refreshing ? 'animate-spin' : ''} />
            Refresh
          </button>
        </div>
        {error && (
          <div role='alert' className='club-preview-inline-error'>
            Some overview data could not be loaded.{' '}
            <button type='button' onClick={onRefresh}>
              Try again
            </button>
          </div>
        )}
        <div className='club-preview-metrics'>
          {metrics.map((metric) => (
            <article className='club-preview-metric' key={metric.key} data-testid={`overview-${metric.key}`}>
              <h4>{metric.title}</h4>
              <div className='club-preview-metric-value'>
                {metric.onClick ? (
                  <button type='button' onClick={metric.onClick} aria-label={`View ${metric.title.toLowerCase()}`}>
                    {metric.value === undefined ? (loading ? '…' : '—') : metric.value.toLocaleString('en')}
                  </button>
                ) : metric.value === undefined ? (
                  loading ? (
                    '…'
                  ) : (
                    '—'
                  )
                ) : (
                  metric.value.toLocaleString('en')
                )}
              </div>
              <p>{metric.caption}</p>
            </article>
          ))}
        </div>
      </section>
    </div>
  )
}
