import { expect, test } from '@playwright/test'

test('Clear all filters removes Pending and reveals an approved booking request', async ({ page }, testInfo) => {
  const requestedStatuses: Array<string | undefined> = []
  const approved = {
    id: 980,
    roomId: '334L',
    requesterId: 1,
    requesterName: 'Local requester',
    priority: 'BOOKING_PRIORITY_INDIVIDUAL',
    title: 'Approved room reservation',
    startTime: '2026-10-14T20:33:00.000Z',
    endTime: '2026-10-14T21:33:00.000Z',
    status: 'BOOKING_REQUEST_STATUS_APPROVED',
    createdAt: '2026-10-07T20:33:00.000Z',
    updatedAt: '2026-10-07T20:33:00.000Z'
  }
  await page.route('**/*', (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = url.pathname.split('/').at(-1) ?? ''
    const headers = {
      'access-control-allow-origin': 'http://localhost:6868',
      'access-control-allow-credentials': 'true'
    }
    if (url.pathname.endsWith('/sessions/whoami')) {
      return route.fulfill({
        headers,
        json: {
          id: 'local-session',
          active: true,
          identity: { id: 'local-identity', traits: { email: 'qa@example.test', name: 'Local QA' } }
        }
      })
    }
    if (method === 'ListBookingRequests') {
      const input = request.postDataJSON() as { status?: string }
      requestedStatuses.push(input.status)
      const bookingRequests = input.status === 'BOOKING_REQUEST_STATUS_PENDING' ? [] : [approved]
      return route.fulfill({ headers, json: { bookingRequests, total: bookingRequests.length } })
    }
    const payloads: Record<string, unknown> = {
      GetMe: {
        user: { id: 1, username: 'local-qa', email: 'qa@example.test' },
        permissions: { isAdmin: true }
      },
      GlobalSearch: { results: [], totalHits: 0 },
      GetBookingConflicts: { conflicts: [] },
      ListOrganizations: { organizations: [], total: 0 },
      ListCategories: { categories: [], total: 0 },
      ListRooms: { rooms: [], total: 0 }
    }
    if (method in payloads) return route.fulfill({ headers, json: payloads[method] })
    if (url.origin === 'http://localhost:6868') return route.continue()
    return route.abort()
  })
  await page.goto('/bookings')
  await expect(page.getByRole('button', { name: 'Status 1', exact: true })).toBeVisible()
  await expect(page.getByText(approved.title, { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: 'Clear all filters', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Status', exact: true })).toBeVisible()
  await expect(page.getByRole('row').filter({ hasText: approved.title })).toContainText('Approved')
  expect(requestedStatuses).toContain('BOOKING_REQUEST_STATUS_PENDING')
  expect(requestedStatuses.at(-1)).toBeUndefined()
  await testInfo.attach('requested-statuses', {
    body: JSON.stringify(requestedStatuses),
    contentType: 'application/json'
  })
  await page.screenshot({ path: testInfo.outputPath('clear-all-shows-approved-request.png') })
})
