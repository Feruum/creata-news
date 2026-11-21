export const ADMIN_OFFLINE_MAX_AGE = 24 * 60 * 60 * 1000

// Deliberate allowlist: adding a new RPC does not silently make it available offline.
const readMethods = {
  'users.v1.UsersService': ['GetMe', 'GetUser', 'ListUsers'],
  'events.v1.EventsService': [
    'GetEvent',
    'GetEventBySlug',
    'ListEvents',
    'ListEventsForAdmin',
    'GetEventPublishReadiness',
    'ListEventCollaborators',
    'ListEventOrganizers'
  ],
  'events.v1.CategoriesService': ['GetCategory', 'ListCategories'],
  'events.v1.OrganizationsService': [
    'GetOrganization',
    'GetOrganizationBySlug',
    'ListOrganizations',
    'GetUserOrganizations',
    'GetOrganizationMembershipHistory',
    'GetUserMembershipHistory',
    'GetUserMembershipTimeline'
  ],
  'events.v1.OrganizationTypesService': ['GetOrganizationType', 'ListOrganizationTypes'],
  'events.v1.OrganizationMembersService': [
    'GetOrganizationMember',
    'ListOrganizationMembers',
    'GetUserOrganizationsWithRole',
    'GetUserCapabilitiesForOrg'
  ],
  'events.v1.RoleTemplatesService': [
    'GetRoleTemplate',
    'ListRoleTemplatesByOrgType',
    'ListRoleTemplatesByOrg',
    'GetTemplateCapabilities'
  ],
  'rooms.v1.RoomsService': [
    'GetRoom',
    'ListRooms',
    'GetRoomAvailability',
    'ListBookingRequests',
    'GetBookingConflicts',
    'GetBookingRequest',
    'ListBookings'
  ],
  'events.v1.EventRegistrationsService': ['GetEventRegistrations', 'GetUserRegistrations'],
  'events.v1.EventAttendanceService': ['GetEventAttendance'],
  'events.v1.StatisticsService': [
    'GetDashboardStatistics',
    'GetEventStatistics',
    'GetEventCategoriesDistributionByMonth',
    'GetEventActivityByYear',
    'GetOverallStatistics',
    'GetEventTrends',
    'GetTopPerformingClubs',
    'GetUserEngagementLevels',
    'GetTopPerformingEvents',
    'GetLowRegistrationEvents',
    'GetOrganizationActivity',
    'GetOrganizationTrends'
  ],
  'activity.v1.ActivityService': ['ListActivityLog', 'ListEntityActivityLog'],
  'search.v1.SearchService': ['GlobalSearch', 'SearchEvents'],
  'schedule.admin': ['ListSubmissions', 'GetSubmission']
}

export const ADMIN_OFFLINE_READ_METHODS = new Set(
  Object.entries(readMethods).flatMap(([service, methods]) => methods.map((method) => `${service}/${method}`))
)

export interface OfflineIdentity {
  identityId: string
  displayName: string
  email: string
  verifiedAt: number
  sessionExpiresAt: number
}

interface SavedRead {
  key: string
  data: string
  updatedAt: number
}

export interface SnapshotDocument {
  version: 1
  epoch: string
  identity: OfflineIdentity
  reads: SavedRead[]
  intentGeneration?: string
}

export interface SnapshotStorage {
  read: () => Promise<SnapshotDocument | null>
  update: (change: (current: SnapshotDocument | null) => SnapshotDocument | null) => Promise<SnapshotDocument | null>
}

export interface IdentityLease {
  identityId: string
  epoch: string
  intentGeneration?: string
}
export interface SnapshotOwner extends IdentityLease {
  verifiedAt: number
}

const sensitiveKey = /token|password|secret|authorization|cookie|credential|oauth|csrf|^session$|^authentication/i

export function serializeSnapshot(value: unknown): string {
  return JSON.stringify(value, (key, item: unknown) => {
    if (sensitiveKey.test(key)) return undefined
    return typeof item === 'bigint' ? { __studyverse_bigint__: item.toString() } : item
  })
}

export function deserializeSnapshot(value: string): unknown {
  return JSON.parse(value, (_key, item: unknown) => {
    if (typeof item === 'object' && item !== null && '__studyverse_bigint__' in item) {
      const bigint = item.__studyverse_bigint__
      if (typeof bigint === 'string' && /^-?\d+$/.test(bigint)) return BigInt(bigint)
    }
    return item
  })
}

function requestKey(method: string, input: unknown): string {
  const canonical = JSON.stringify(input, (key, item: unknown) => {
    if (sensitiveKey.test(key)) return undefined
    if (typeof item === 'bigint') return { __studyverse_bigint__: item.toString() }
    if (item && typeof item === 'object' && !Array.isArray(item)) {
      return Object.fromEntries(Object.entries(item).sort(([a], [b]) => a.localeCompare(b)))
    }
    return item
  })
  return `${method}:${canonical}`
}

export function snapshotExpiresAt(identity: OfflineIdentity): number {
  return Math.min(identity.verifiedAt + ADMIN_OFFLINE_MAX_AGE, identity.sessionExpiresAt)
}

export class OfflineSnapshots {
  identity: OfflineIdentity | null = null
  storageAvailable = true
  private epoch = ''
  private revision = 0
  private intentGeneration: string | undefined
  private readonly storage: SnapshotStorage
  private readonly now: () => number

  constructor(storage: SnapshotStorage, now: () => number = Date.now) {
    this.storage = storage
    this.now = now
  }

  lease(): IdentityLease | null {
    return this.identity
      ? { identityId: this.identity.identityId, epoch: this.epoch, intentGeneration: this.intentGeneration }
      : null
  }

  isCurrent(lease: IdentityLease | null): boolean {
    return (
      !!lease &&
      lease.identityId === this.identity?.identityId &&
      lease.epoch === this.epoch &&
      lease.intentGeneration === this.intentGeneration
    )
  }

  invalidate(): void {
    this.revision += 1
    this.identity = null
    this.epoch = ''
    this.intentGeneration = undefined
  }

  async verify(
    identity: OfflineIdentity,
    expectedOwner?: SnapshotOwner | null,
    intentGeneration?: string
  ): Promise<boolean> {
    const revision = ++this.revision
    this.identity = identity
    this.epoch = `${this.now()}-${Math.random().toString(36).slice(2)}`
    this.intentGeneration = intentGeneration
    let accepted = true
    try {
      const saved = await this.storage.update((current) => {
        if (revision !== this.revision) return current
        if (
          expectedOwner !== undefined &&
          current &&
          current.identity.identityId !== identity.identityId &&
          (current.epoch !== expectedOwner?.epoch || current.identity.verifiedAt !== expectedOwner?.verifiedAt)
        ) {
          accepted = false
          return current
        }
        if (!Number.isFinite(identity.sessionExpiresAt)) return null
        return {
          version: 1,
          epoch:
            current?.identity.identityId === identity.identityId &&
            current.identity.sessionExpiresAt === identity.sessionExpiresAt &&
            current.intentGeneration === intentGeneration
              ? current.epoch
              : this.epoch,
          identity,
          intentGeneration,
          reads: current?.identity.identityId === identity.identityId ? current.reads : []
        }
      })
      if (revision === this.revision && saved && accepted) this.epoch = saved.epoch
      if (!accepted && revision === this.revision) this.invalidate()
    } catch {
      if (revision === this.revision) this.storageAvailable = false
    }
    return accepted && revision === this.revision
  }

  async restore(): Promise<OfflineIdentity | null> {
    const revision = ++this.revision
    try {
      const saved = await this.storage.read()
      if (revision !== this.revision) return null
      if (!saved) {
        this.invalidate()
        return null
      }
      if (saved.version !== 1 || snapshotExpiresAt(saved.identity) <= this.now()) {
        await this.clearOwner({
          identityId: saved.identity.identityId,
          epoch: saved.epoch,
          verifiedAt: saved.identity.verifiedAt
        })
        return null
      }
      this.identity = saved.identity
      this.epoch = saved.epoch
      this.intentGeneration = saved.intentGeneration
      return saved.identity
    } catch {
      if (revision !== this.revision) return null
      this.storageAvailable = false
      this.invalidate()
      return null
    }
  }

  async clear(): Promise<OfflineIdentity | null> {
    let removed = this.identity
    this.invalidate()
    const revision = this.revision
    try {
      await this.storage.update((current) => {
        if (revision !== this.revision) return current
        removed = current?.identity ?? removed
        return null
      })
    } catch {
      this.storageAvailable = false
    }
    return removed
  }

  async clearIdentity(identityId: string, intentGeneration?: string): Promise<void> {
    if (
      this.identity?.identityId === identityId &&
      (intentGeneration === undefined ||
        this.intentGeneration === undefined ||
        this.intentGeneration === intentGeneration)
    )
      this.invalidate()
    try {
      await this.storage.update((current) =>
        current?.identity.identityId === identityId &&
        (intentGeneration === undefined ||
          current.intentGeneration === undefined ||
          current.intentGeneration === intentGeneration)
          ? null
          : current
      )
    } catch {
      this.storageAvailable = false
    }
  }

  async captureOwner(readPersisted = false): Promise<SnapshotOwner | null> {
    const lease = this.lease()
    if (!readPersisted && lease && this.identity) return { ...lease, verifiedAt: this.identity.verifiedAt }
    try {
      const saved = await this.storage.read()
      return saved
        ? { identityId: saved.identity.identityId, epoch: saved.epoch, verifiedAt: saved.identity.verifiedAt }
        : null
    } catch {
      this.storageAvailable = false
      return null
    }
  }

  async clearOwner(owner: SnapshotOwner): Promise<boolean> {
    if (
      this.identity?.identityId === owner.identityId &&
      this.epoch === owner.epoch &&
      this.identity.verifiedAt === owner.verifiedAt
    )
      this.invalidate()
    let removed = false
    try {
      await this.storage.update((current) => {
        if (
          current?.identity.identityId !== owner.identityId ||
          current.epoch !== owner.epoch ||
          current.identity.verifiedAt !== owner.verifiedAt
        )
          return current
        removed = true
        return null
      })
    } catch {
      this.storageAvailable = false
    }
    return removed
  }

  async save(lease: IdentityLease | null, method: string, input: unknown, data: unknown): Promise<boolean> {
    if (
      !this.isCurrent(lease) ||
      !ADMIN_OFFLINE_READ_METHODS.has(method) ||
      !this.storageAvailable ||
      !Number.isFinite(this.identity?.sessionExpiresAt)
    )
      return false
    const key = requestKey(method, input)
    let accepted = false
    try {
      await this.storage.update((current) => {
        if (
          !lease ||
          !current ||
          !this.isCurrent(lease) ||
          current.epoch !== lease.epoch ||
          current.intentGeneration !== lease.intentGeneration ||
          current.identity.identityId !== lease.identityId
        )
          return current
        const now = this.now()
        if (snapshotExpiresAt(current.identity) <= now) return null
        accepted = true
        return {
          ...current,
          reads: [
            ...current.reads
              .filter((read) => read.key !== key && read.updatedAt + ADMIN_OFFLINE_MAX_AGE > now)
              .slice(-199),
            { key, data: serializeSnapshot(data), updatedAt: now }
          ]
        }
      })
    } catch {
      this.storageAvailable = false
      accepted = false
    }
    return accepted
  }

  async read(method: string, input: unknown): Promise<{ data: unknown; updatedAt: number } | null> {
    if (!this.identity || !ADMIN_OFFLINE_READ_METHODS.has(method) || !this.storageAvailable) return null
    const lease = this.lease()
    try {
      const saved = await this.storage.read()
      if (
        !lease ||
        !saved ||
        !this.isCurrent(lease) ||
        saved.epoch !== lease.epoch ||
        saved.intentGeneration !== lease.intentGeneration ||
        saved.identity.identityId !== lease.identityId
      )
        return null
      if (snapshotExpiresAt(saved.identity) <= this.now()) {
        await this.clearOwner({
          identityId: saved.identity.identityId,
          epoch: saved.epoch,
          verifiedAt: saved.identity.verifiedAt
        })
        return null
      }
      const read = saved.reads.find((item) => item.key === requestKey(method, input))
      if (!read || read.updatedAt + ADMIN_OFFLINE_MAX_AGE <= this.now()) return null
      return { data: deserializeSnapshot(read.data), updatedAt: read.updatedAt }
    } catch {
      this.storageAvailable = false
      return null
    }
  }
}
