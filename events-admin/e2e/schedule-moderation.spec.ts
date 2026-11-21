import { expect, type Page, test } from '@playwright/test'
import type { SubmissionSummary } from '../src/features/schedules'

const endpoint = '/api/schedule/admin/submissions'
const settings = { studyYear: 2026, term: 1, firstWeekStart: '2026-09-07' }
const oldLesson = {
  id: 'old',
  title: 'Published lecture',
  start: '2026-10-05T04:00:00.000Z',
  end: '2026-10-05T05:00:00.000Z',
  teacher: 'Published teacher',
  lessonType: 'Lecture',
  location: 'C1.101',
  online: false
}
const newLesson = {
  ...oldLesson,
  id: 'new',
  title: 'New lecture',
  teacher: 'New teacher',
  lessonType: 'Practice',
  location: 'Teams',
  online: true
}
const unchangedLesson = { ...oldLesson, id: 'unchanged', title: 'Shared lab' }
const pending: SubmissionSummary = {
  id: 'submission-1',
  groupName: 'SE-2401',
  authorId: 'author-123',
  contentHash: 'hash-1',
  status: 'pending',
  createdAt: '2026-10-04T05:00:00.000Z',
  reviewedAt: null,
  reviewerId: null,
  rejectionReason: null
}

async function mockAuth(page: Page, isAdmin = true) {
  const cors = {
    'access-control-allow-origin': 'http://localhost:6868',
    'access-control-allow-credentials': 'true'
  }
  await page.route('**/sessions/whoami', (route) =>
    route.fulfill({
      json: {
        id: 'test-session',
        active: true,
        identity: { id: 'admin-identity', traits: { email: 'admin@example.test', name: 'Admin' } }
      },
      headers: cors
    })
  )
  await page.route('**/users.v1.UsersService/GetMe', (route) =>
    route.fulfill({
      json: {
        user: { id: 1, username: 'reviewer', email: 'admin@example.test' },
        permissions: { isAdmin, isStaff: !isAdmin, managedClubIds: [4] }
      },
      headers: cors
    })
  )
  await page.route('**/search.v1.SearchService/GlobalSearch', (route) =>
    route.fulfill({ json: { results: [], totalHits: 0 }, headers: cors })
  )
  await page.route('**/ListOrganizations', (route) =>
    route.fulfill({ json: { organizations: [], total: 0 }, headers: cors })
  )
  await page.route('**/ListCategories', (route) => route.fulfill({ json: { categories: [], total: 0 }, headers: cors }))
  await page.route('**/ListRooms', (route) => route.fulfill({ json: { rooms: [], total: 0 }, headers: cors }))
}

async function mockSchedules(page: Page, options: { conflict?: boolean; unavailable?: boolean } = {}) {
  let submission = { ...pending }
  let postCount = 0
  const requests: { id: string; decision: string; reason?: string }[] = []
  const listRequests: URL[] = []
  await page.route(`**${endpoint}**`, (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (request.method() === 'POST') {
      postCount += 1
      const body = request.postDataJSON() as { id: string; decision: string; reason?: string }
      requests.push(body)
      submission = {
        ...submission,
        status: body.decision === 'approve' ? 'approved' : 'rejected',
        reviewedAt: '2026-10-04T06:00:00.000Z',
        reviewerId: 'reviewer-456',
        rejectionReason: body.reason ?? null
      }
      if (options.conflict) {
        submission.status = 'approved'
        return route.fulfill({ status: 409, json: { error: 'Submission already reviewed' } })
      }
      return route.fulfill({ json: submission })
    }
    if (options.unavailable) return route.fulfill({ status: 403, json: { error: 'Forbidden' } })
    if (url.searchParams.has('id')) {
      return route.fulfill({
        json: {
          submission: { ...submission, settings, events: [newLesson, unchangedLesson] },
          published: {
            groupName: 'SE-2401',
            settings: { ...settings, term: 2 },
            events: [oldLesson, unchangedLesson],
            updatedAt: '2026-10-01T05:00:00.000Z'
          },
          diff: { added: [newLesson], removed: [oldLesson] }
        }
      })
    }
    listRequests.push(url)
    return route.fulfill({
      json: {
        submissions:
          url.searchParams.get('status') === 'pending' && submission.status !== 'pending' ? [] : [submission],
        nextCursor: null
      }
    })
  })
  return { requests, listRequests, getPostCount: () => postCount }
}

test.beforeEach(async ({ page }) => {
  await page.route('**/*', (route) => {
    if (new URL(route.request().url()).origin !== 'http://localhost:6868') return route.abort()
    return route.continue()
  })
  await mockAuth(page)
})

test('admin queue shows the complete snapshot and added/removed lesson details', async ({ page }) => {
  await mockSchedules(page)
  await page.goto('/schedules')
  await expect(page.getByRole('heading', { name: 'Schedule moderation', exact: true })).toBeVisible()
  await expect(page.locator('a[data-sidebar="menu-button"]').filter({ hasText: 'Schedules' })).toBeVisible()
  await page.getByRole('button', { name: 'Review SE-2401', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByText('author-123', { exact: true })).toBeVisible()
  const added = dialog.getByRole('table', { name: 'Added lessons', exact: true })
  const removed = dialog.getByRole('table', { name: 'Removed lessons', exact: true })
  const snapshot = dialog.getByRole('table', { name: 'Submitted lessons', exact: true })
  await expect(added).toContainText('New lecture')
  await expect(added).toContainText('New teacher')
  await expect(added).toContainText('Practice')
  await expect(added).toContainText('Teams')
  await expect(added).toContainText('Online')
  await expect(removed).toContainText('Published lecture')
  await expect(snapshot).toContainText('New lecture')
  await expect(snapshot).toContainText('Shared lab')
  await expect(snapshot.getByRole('row')).toHaveCount(3)
  await expect(dialog.getByText('Schedule settings changed.', { exact: true })).toBeVisible()
})

test('approval publishes once and moves the decision into history', async ({ page }) => {
  const mock = await mockSchedules(page)
  await page.goto('/schedules')
  await page.getByRole('button', { name: 'Review SE-2401', exact: true }).click()
  await page.getByRole('button', { name: 'Approve', exact: true }).click()
  await expect(page.getByRole('alertdialog')).toContainText('complete submitted schedule')
  await page.getByRole('button', { name: 'Approve and publish', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.getByText('No pending schedules', { exact: true })).toBeVisible()
  expect(mock.requests).toEqual([{ id: 'submission-1', decision: 'approve' }])
  await page.getByRole('tab', { name: 'History', exact: true }).click()
  await expect(page.getByRole('table', { name: 'Schedule submissions', exact: true })).toContainText('Approved')
  await expect(page.getByText('reviewer-456', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'View SE-2401', exact: true }).click()
  await expect(page.getByRole('dialog').getByRole('button', { name: 'Approve', exact: true })).toHaveCount(0)
})

test('rejection requires a nonblank reason and records the trimmed reason in history', async ({ page }) => {
  const mock = await mockSchedules(page)
  await page.goto('/schedules')
  await page.getByRole('button', { name: 'Review SE-2401', exact: true }).click()
  const reject = page.getByRole('button', { name: 'Reject', exact: true })
  await expect(reject).toBeDisabled()
  await page.getByLabel('Rejection reason', { exact: true }).fill('   ')
  await expect(reject).toBeDisabled()
  expect(mock.getPostCount()).toBe(0)
  await page.getByLabel('Rejection reason', { exact: true }).fill('  Lesson room is incorrect  ')
  await reject.click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  expect(mock.requests).toEqual([{ id: 'submission-1', decision: 'reject', reason: 'Lesson room is incorrect' }])
  await page.getByRole('tab', { name: 'History', exact: true }).click()
  await expect(page.getByRole('table', { name: 'Schedule submissions', exact: true })).toContainText('Rejected')
  await expect(page.getByText('Lesson room is incorrect', { exact: true })).toBeVisible()
})

test('a conflicting review refreshes the record and blocks further decisions', async ({ page }) => {
  const mock = await mockSchedules(page, { conflict: true })
  await page.goto('/schedules')
  await page.getByRole('button', { name: 'Review SE-2401', exact: true }).click()
  await page.getByLabel('Rejection reason', { exact: true }).fill('Wrong dates')
  await page.getByRole('button', { name: 'Reject', exact: true }).click()
  await expect(page.getByRole('dialog').getByRole('alert')).toContainText('already been reviewed')
  await expect(page.getByRole('dialog').getByText('Approved', { exact: true })).toBeVisible()
  await expect(page.getByRole('dialog').getByRole('button', { name: 'Approve', exact: true })).toHaveCount(0)
  await expect(page.getByRole('dialog').getByRole('button', { name: 'Reject', exact: true })).toHaveCount(0)
  expect(mock.getPostCount()).toBe(1)
})

test('staff and organization managers cannot open or discover schedule moderation', async ({ page }) => {
  await mockAuth(page, false)
  const mock = await mockSchedules(page)
  await page.goto('/schedules')
  await expect(page.getByRole('heading', { name: 'Admin access required', exact: true })).toBeVisible()
  await expect(page.locator('a[data-sidebar="menu-button"]').filter({ hasText: 'Schedules' })).toHaveCount(0)
  await page.getByRole('button', { name: 'Open search (⌘K)', exact: true }).click()
  await page.getByPlaceholder('Type a command or search...').fill('schedule')
  await expect(page.getByRole('option', { name: /Schedules/ })).toHaveCount(0)
  expect(mock.listRequests).toHaveLength(0)
})

test('history and group filtering use the root API endpoint', async ({ page }) => {
  const mock = await mockSchedules(page)
  await page.goto('/schedules')
  await page.getByRole('tab', { name: 'History', exact: true }).click()
  await page.getByLabel('Group', { exact: true }).fill('SE-2401')
  await page.getByRole('button', { name: 'Filter', exact: true }).click()
  await expect.poll(() => mock.listRequests.at(-1)?.searchParams.get('group')).toBe('SE-2401')
  expect(mock.listRequests.at(-1)?.searchParams.get('status')).toBe('all')
  expect(mock.listRequests.every((url) => url.pathname === endpoint)).toBe(true)
})

test('cursor pagination retains previous submissions and opens reviewed history without actions', async ({ page }) => {
  const requestedCursors: (string | null)[] = []
  await page.route(`**${endpoint}**`, (route) => {
    const url = new URL(route.request().url())
    const reviewed = {
      ...pending,
      id: 'submission-2',
      groupName: 'SE-2402',
      status: 'rejected',
      rejectionReason: 'Wrong semester',
      reviewedAt: '2026-10-04T06:00:00.000Z',
      reviewerId: 'reviewer-456'
    }
    const superseded = { ...pending, id: 'submission-3', groupName: 'SE-2403', status: 'superseded' }
    if (url.searchParams.has('id')) {
      return route.fulfill({
        json: {
          submission: { ...reviewed, settings, events: [oldLesson] },
          published: null,
          diff: { added: [oldLesson], removed: [] }
        }
      })
    }
    requestedCursors.push(url.searchParams.get('cursor'))
    return route.fulfill({
      json: url.searchParams.has('cursor')
        ? { submissions: [reviewed, superseded], nextCursor: null }
        : { submissions: [pending], nextCursor: 'cursor+two' }
    })
  })
  await page.goto('/schedules')
  await page.getByRole('tab', { name: 'History', exact: true }).click()
  await page.getByRole('button', { name: 'Load more submissions', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Review SE-2401', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'View SE-2402', exact: true })).toBeVisible()
  await expect(page.getByRole('row').filter({ hasText: 'SE-2403' })).toContainText('Replaced by a newer submission')
  await expect(page.getByRole('button', { name: 'Load more submissions', exact: true })).toHaveCount(0)
  expect(requestedCursors).toContain('cursor+two')
  await page.getByRole('button', { name: 'View SE-2402', exact: true }).click()
  await expect(page.getByRole('dialog').getByText('Wrong semester', { exact: true })).toBeVisible()
  await expect(page.getByRole('dialog').getByRole('button', { name: 'Reject', exact: true })).toHaveCount(0)
  await expect(page.getByRole('dialog').getByRole('button', { name: 'Approve', exact: true })).toHaveCount(0)
})

test('administrators can discover moderation with keyboard search', async ({ page }) => {
  await mockSchedules(page)
  await page.goto('/schedules')
  await expect(page.getByRole('heading', { name: 'Schedule moderation', exact: true })).toBeVisible()
  await page.keyboard.press('Control+k')
  await page.getByPlaceholder('Type a command or search...').fill('schedules')
  await page.getByRole('option', { name: 'Schedules G S', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.getByRole('heading', { name: 'Schedule moderation', exact: true })).toBeVisible()
})

test('forbidden API response shows access recovery instead of a queue', async ({ page }) => {
  await mockSchedules(page, { unavailable: true })
  await page.goto('/schedules')
  await expect(page.getByRole('alert')).toContainText('Admin access is required')
  await expect(page.getByRole('button', { name: 'Review SE-2401', exact: true })).toHaveCount(0)
})

test('mobile review keeps lessons reachable and restores focus after Escape', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await mockSchedules(page)
  await page.goto('/schedules')
  const trigger = page.getByRole('button', { name: 'Review SE-2401', exact: true })
  await trigger.click()
  await expect(page.getByRole('dialog').getByRole('table', { name: 'Submitted lessons', exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(trigger).toBeFocused()
})
