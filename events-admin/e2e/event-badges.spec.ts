import { expect, type Page, test } from '@playwright/test'

test.setTimeout(60_000)

const badgeStatuses = {
  none: 'EVENT_BADGE_STATUS_NOT_REQUESTED',
  pending: 'EVENT_BADGE_STATUS_PENDING',
  approved: 'EVENT_BADGE_STATUS_APPROVED',
  rejected: 'EVENT_BADGE_STATUS_REJECTED'
} as const

type Badge = { status: string; reviewedBy?: number; reviewedAt?: string }

const organization = { id: 1, title: 'Test student organization', slug: 'test-host' }
const seedEvent = {
  id: 902,
  title: 'Campus community workshop',
  description: JSON.stringify([{ type: 'p', children: [{ text: 'Synthetic event for badge workflow testing.' }] }]),
  userId: 1,
  organizationId: 1,
  organization,
  startTime: '2026-10-14T12:00:00Z',
  endTime: '2026-10-14T14:00:00Z',
  createdAt: '2026-10-13T00:00:00Z',
  updatedAt: '2026-10-13T00:00:00Z',
  format: 'EVENT_FORMAT_OFFLINE',
  status: 'EVENT_STATUS_PUBLISHED',
  structuredLocation: { type: 'LOCATION_TYPE_EXTERNAL', externalAddress: 'Campus community hall' },
  slug: 'campus-community-workshop',
  socGpaBadge: { status: badgeStatuses.pending } as Badge,
  irosBadge: { status: badgeStatuses.pending } as Badge
}

async function mockBadges(
  page: Page,
  options: { administrator?: boolean; empty?: boolean; failReview?: boolean } = {}
) {
  let event = { ...seedEvent }
  let exists = !options.empty
  let failReview = options.failReview ?? false
  const creates: Record<string, unknown>[] = []
  const updates: Record<string, unknown>[] = []
  const reviews: Record<string, unknown>[] = []

  await page.route('**/*', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = url.pathname.split('/').at(-1) ?? ''
    const headers = {
      'access-control-allow-origin': new URL(page.url() === 'about:blank' ? 'http://localhost:6868' : page.url())
        .origin,
      'access-control-allow-credentials': 'true'
    }
    if (url.pathname.endsWith('/sessions/whoami')) {
      return route.fulfill({
        headers,
        json: {
          id: 'badge-test-session',
          active: true,
          identity: { id: 'badge-test-identity', traits: { email: 'badges@example.test', name: 'Badge QA' } }
        }
      })
    }
    if (method === 'CreateEvent') {
      const body: Record<string, unknown> = request.postDataJSON()
      creates.push(body)
      exists = true
      event = {
        ...event,
        title: String(body.title),
        description: String(body.description),
        socGpaBadge: { status: body.socGpaRequested ? badgeStatuses.pending : badgeStatuses.none },
        irosBadge: { status: body.irosRequested ? badgeStatuses.pending : badgeStatuses.none }
      }
      return route.fulfill({ headers, json: { event } })
    }
    if (method === 'UpdateEvent') {
      const body: Record<string, unknown> = request.postDataJSON()
      updates.push(body)
      for (const [flag, field] of [
        ['socGpaRequested', 'socGpaBadge'],
        ['irosRequested', 'irosBadge']
      ] as const) {
        if (body[flag] !== undefined) {
          event[field] = body[flag]
            ? event[field].status === badgeStatuses.approved
              ? event[field]
              : { status: badgeStatuses.pending }
            : { status: badgeStatuses.none }
        }
      }
      return route.fulfill({ headers, json: { event } })
    }
    if (method === 'ReviewEventBadges') {
      const body: Record<string, unknown> = request.postDataJSON()
      reviews.push(body)
      if (!options.administrator) {
        return route.fulfill({ status: 403, headers, json: { code: 'permission_denied', message: 'Admin required' } })
      }
      if (failReview) {
        return route.fulfill({ status: 503, headers, json: { code: 'unavailable', message: 'Review unavailable' } })
      }
      for (const [flag, field] of [
        ['socGpaStatus', 'socGpaBadge'],
        ['irosStatus', 'irosBadge']
      ] as const) {
        if (body[flag] !== undefined) {
          event[field] = { status: String(body[flag]), reviewedBy: 99, reviewedAt: new Date().toISOString() }
        }
      }
      return route.fulfill({ headers, json: { event } })
    }
    const payloads: Record<string, unknown> = {
      GetMe: {
        user: { id: 1, username: 'badge-qa', email: 'badges@example.test' },
        permissions: { isAdmin: options.administrator ?? false, managedClubIds: [1] }
      },
      GlobalSearch: { results: [], totalHits: 0 },
      ListEventsForAdmin: { events: exists ? [event] : [], total: exists ? 1 : 0 },
      GetEvent: {
        event,
        canEdit: true,
        canDelete: false,
        canManageRegistrations: false,
        canReviewBadges: !!options.administrator
      },
      ListOrganizations: { organizations: [organization], total: 1 },
      ListCategories: { categories: [], total: 0 },
      ListRooms: { rooms: [], total: 0 }
    }
    if (method in payloads) return route.fulfill({ headers, json: payloads[method] })
    if (url.origin === 'http://localhost:6868') return route.continue()
    return route.abort()
  })
  return {
    creates,
    updates,
    reviews,
    allowReview: () => {
      failReview = false
    }
  }
}

for (const width of [1280, 390]) {
  test.describe(`event badges at ${width}px`, () => {
    test.use({ viewport: { width, height: 900 }, locale: 'en-US' })

    test('ordinary event creation requests both badges without granting approval', async ({ page }, testInfo) => {
      const api = await mockBadges(page, { empty: true })
      await page.goto('/events?action=create')
      const dialog = page.getByRole('dialog', { name: 'Create Event', exact: true })
      await expect(dialog.getByRole('checkbox', { name: 'Request SocGpa' })).not.toBeChecked()
      await expect(dialog.getByRole('checkbox', { name: 'Request iRos' })).not.toBeChecked()
      await dialog.getByRole('textbox', { name: 'Title', exact: true }).fill('Campus community workshop')
      await dialog.getByRole('button', { name: 'Description', exact: true }).click()
      const description = page.getByRole('dialog', { name: 'Описание мероприятия' })
      await description.locator('[contenteditable="true"]').fill('Synthetic event for badge workflow testing.')
      await description.getByRole('button', { name: 'Сохранить', exact: true }).click()
      await dialog.getByRole('button', { name: 'External Venue', exact: true }).click()
      await dialog.getByRole('textbox', { name: 'Address', exact: true }).fill('Campus community hall')
      await dialog.getByRole('combobox').filter({ hasText: 'Select primary host' }).click()
      await page.getByRole('option').filter({ hasText: organization.title }).click()
      await dialog.getByRole('switch', { name: 'Include time' }).uncheck()
      const tomorrow = new Date()
      tomorrow.setDate(tomorrow.getDate() + 1)
      const dateLabel = new RegExp(
        `${tomorrow.toLocaleDateString('en-US', { month: 'long' })} ${tomorrow.getDate()}(?:st|nd|rd|th), ${tomorrow.getFullYear()}$`
      )
      for (let i = 0; i < 2; i++) {
        await dialog.getByRole('button', { name: 'Pick date…', exact: true }).first().click()
        await page.getByRole('button', { name: dateLabel }).click()
        await page.keyboard.press('Escape')
        await expect(
          page.getByRole('grid', {
            name: tomorrow.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
          })
        ).not.toBeVisible()
      }
      await dialog.getByRole('checkbox', { name: 'Request SocGpa' }).check()
      await dialog.getByRole('checkbox', { name: 'Request iRos' }).check()
      await page.screenshot({ path: testInfo.outputPath('create-badge-requests.png') })
      await dialog.getByRole('button', { name: 'Create Event', exact: true }).click()
      await expect(dialog).not.toBeVisible()
      expect(api.creates).toHaveLength(1)
      expect(api.creates[0]).toMatchObject({ socGpaRequested: true, irosRequested: true })
      await page.goto('/events?event=902')
      const approval = page.getByRole('region', { name: 'Event badge approval' })
      await expect(approval).toContainText('Pending approval')
      await expect(approval.getByRole('button')).toHaveCount(0)
      await page.screenshot({ path: testInfo.outputPath('editor-pending-badges.png') })
    })

    test('administrator reviews independently and edit forms use the latest decisions', async ({ page }, testInfo) => {
      const api = await mockBadges(page, { administrator: true })
      await page.goto('/events?event=902')
      const details = page.getByRole('dialog', { name: seedEvent.title, exact: true })
      await details.getByRole('button', { name: 'Edit event', exact: true }).click()
      let edit = page.getByRole('dialog', { name: 'Edit Event', exact: true })
      await expect(edit.getByRole('checkbox', { name: 'Request SocGpa' })).toBeChecked()
      await edit.getByRole('button', { name: 'Cancel', exact: true }).click()
      await details.getByRole('button', { name: 'Reject SocGpa', exact: true }).click()
      await expect(details.getByText('SocGpa · Rejected', { exact: true })).toBeVisible()
      await details.getByRole('button', { name: 'Edit event', exact: true }).click()
      edit = page.getByRole('dialog', { name: 'Edit Event', exact: true })
      await expect(edit.getByRole('checkbox', { name: 'Request SocGpa' })).not.toBeChecked()
      await expect(edit.getByRole('checkbox', { name: 'Request iRos' })).toBeChecked()
      await edit.getByRole('checkbox', { name: 'Request SocGpa' }).check()
      await edit.getByRole('button', { name: 'Save Changes', exact: true }).click()
      await expect(edit).not.toBeVisible()
      expect(api.updates.at(-1)).toMatchObject({ socGpaRequested: true })
      expect(api.updates.at(-1)).not.toHaveProperty('irosRequested')
      await expect(details.getByText('SocGpa · Pending approval', { exact: true })).toBeVisible()
      await page.screenshot({ path: testInfo.outputPath('administrator-badge-review.png') })
      await details.getByRole('button', { name: 'Approve SocGpa', exact: true }).click()
      await expect(details.getByText('SocGpa', { exact: true }).first()).toBeVisible()
      await expect(details.getByRole('button', { name: 'Remove SocGpa approval' })).toBeVisible()
      await expect(details.getByRole('button', { name: 'Approve iRos' })).toBeVisible()
      expect(api.reviews.at(-1)).toEqual({ eventId: 902, socGpaStatus: badgeStatuses.approved })
      await details.getByRole('button', { name: 'Edit event', exact: true }).click()
      edit = page.getByRole('dialog', { name: 'Edit Event', exact: true })
      await expect(edit.getByRole('checkbox', { name: 'Request SocGpa' })).toBeChecked()
      await edit.getByRole('button', { name: 'Save Changes', exact: true }).click()
      await expect(edit).not.toBeVisible()
      expect(api.updates.at(-1)).not.toHaveProperty('socGpaRequested')
      expect(api.updates.at(-1)).not.toHaveProperty('irosRequested')
      await expect(details.getByRole('button', { name: 'Remove SocGpa approval' })).toBeVisible()
      await details.getByRole('button', { name: 'Approve iRos', exact: true }).click()
      await expect(details.getByRole('button', { name: 'Remove iRos approval' })).toBeVisible()
      await page.screenshot({ path: testInfo.outputPath('approved-event-badges.png') })
    })
  })
}

test('failed review keeps requests pending and offers a working retry', async ({ page }) => {
  const api = await mockBadges(page, { administrator: true, failReview: true })
  await page.goto('/events?event=902')
  const details = page.getByRole('dialog', { name: seedEvent.title, exact: true })
  const approve = details.getByRole('button', { name: 'Approve SocGpa', exact: true })
  await approve.click()
  await expect(details.getByRole('alert')).toHaveText('Could not save badge review. Please try again.')
  await expect(approve).toBeEnabled()
  await expect(details.getByText('SocGpa · Pending approval', { exact: true })).toBeVisible()
  api.allowReview()
  await approve.click()
  await expect(details.getByRole('alert')).toHaveCount(0)
  await expect(details.getByRole('button', { name: 'Remove SocGpa approval' })).toBeVisible()
  expect(api.reviews).toHaveLength(2)
})
